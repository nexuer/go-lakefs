package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/nexuer/go-lakefs"
)

func main() {
	mux := http.NewServeMux()
	lakefsClient := lakefs.NewClient(lakefs.BasicAuth{
		Endpoint:        os.Getenv("LAKEFS_ENDPOINT"),
		AccessKeyID:     os.Getenv("LAKEFS_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("LAKEFS_SECRET_ACCESS_KEY"),
	}, &lakefs.Options{
		Debug: true,
	})

	handler(mux, lakefsClient)

	server := &http.Server{
		Addr:    ":8080",
		Handler: corsMiddleware(mux),
	}

	go func() {
		slog.Info(fmt.Sprintf("server listening at %s", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error(fmt.Sprintf("error listening at %s", server.Addr))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info(fmt.Sprintf("got signal %s, shutting down...", sig))

	// 设置 5 秒超时优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Info(fmt.Sprintf("error shutting down: %s", err.Error()))
	}

	slog.Info(fmt.Sprintf("shutdown complete"))
}

type Error struct {
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, err error, statusCode ...int) {
	if err == nil {
		return
	}
	code := http.StatusInternalServerError
	if len(statusCode) > 0 {
		code = statusCode[0]
	}
	w.WriteHeader(code)
	writeJson(w, Error{Message: err.Error()})
}

func writeJson(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	d, _ := json.Marshal(data)
	_, _ = w.Write(d)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 无论什么请求，都先设置 CORS header
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept")
		w.Header().Set("Access-Control-Max-Age", "86400") // 预检缓存 24 小时

		// 如果是预检请求（OPTIONS），直接返回 204
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// 继续处理实际请求
		next.ServeHTTP(w, r)
	})
}

func handler(mux *http.ServeMux, lakefsClient *lakefs.Client) {
	// 查询苍仓库列表
	mux.HandleFunc("GET /api/v1/repos", func(writer http.ResponseWriter, request *http.Request) {
		ctx := context.Background()
		search := request.URL.Query().Get("search")
		repos, err := lakefsClient.Repositories.ListRepositories(ctx, &lakefs.ListRepositoriesOptions{
			Search: search,
		})
		if err != nil {
			writeError(writer, err)
			return
		}
		writeJson(writer, repos.Results)

	})

	// 创建仓库
	mux.HandleFunc("POST /api/v1/repos", func(writer http.ResponseWriter, request *http.Request) {
		ctx := context.Background()
		var repositoryCreation lakefs.RepositoryCreation
		if err := json.NewDecoder(request.Body).Decode(&repositoryCreation); err != nil {
			writeError(writer, err)
			return
		}
		// 这个是必须的，现在固定本地
		repositoryCreation.StorageNamespace = fmt.Sprintf("s3://lakefs-data/%s/", repositoryCreation.Name)
		repo, err := lakefsClient.Repositories.CreateRepository(ctx, &repositoryCreation)
		if err != nil {
			writeError(writer, err)
			return
		}
		writeJson(writer, repo)
	})

	// 删除仓库
	mux.HandleFunc("DELETE /api/v1/repos/{id}", func(writer http.ResponseWriter, request *http.Request) {
		id := request.PathValue("id")
		if id == "" {
			writeError(writer, errors.New("id is required"), http.StatusBadRequest)
			return
		}
		ctx := context.Background()
		err := lakefsClient.Repositories.DeleteRepository(ctx, id)
		if err != nil {
			writeError(writer, err)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	})

	// 获取上传地址, 先固定都上传到main分支
	mux.HandleFunc("GET /api/v1/repos/{id}/branches/main/presign/{filename}", func(writer http.ResponseWriter, request *http.Request) {
		id := request.PathValue("id")
		filename := request.PathValue("filename")
		fmt.Printf("id: %s, filename: %s\n", id, filename)
		ctx := context.Background()
		data, err := lakefsClient.Staging.GetBacking(ctx, id, "main", &lakefs.GetBackingOptions{
			Path:    filename,
			Presign: true,
		})
		if err != nil {
			writeError(writer, err)
			return
		}
		writeJson(writer, data)
	})

	// 关联上传的文件到仓库分支, 先固定关联到main分支
	mux.HandleFunc("POST /api/v1/repos/{id}/branches/main/files", func(writer http.ResponseWriter, request *http.Request) {
		id := request.PathValue("id")
		var files PostFilesRequest
		if err := json.NewDecoder(request.Body).Decode(&files); err != nil {
			writeError(writer, err)
			return
		}
		ctx := context.Background()
		errs := make([]string, 0, len(files.Files))

		for _, file := range files.Files {
			filename := file.UserMetadata["filename"]
			_, err := lakefsClient.Staging.PutBacking(ctx, id, "main", &lakefs.PutBackingOptions{
				Path:            filename,
				StagingMetadata: file,
			})
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: %s", filename, err.Error()))
			}
		}

		if len(errs) > 0 {
			writeError(writer, errors.New(strings.Join(errs, ";")))
			return
		}
		writeJson(writer, &PostFilesResponse{
			Total: len(files.Files),
		})
	})

	// 返回文件列表
	mux.HandleFunc("GET /api/v1/repos/{id}/branches/main/files", func(writer http.ResponseWriter, request *http.Request) {
		id := request.PathValue("id")
		ctx := context.Background()
		objects, err := lakefsClient.Objects.ListObjects(ctx, id, "main", &lakefs.ListObjectOptions{})
		if err != nil {
			writeError(writer, err)
			return
		}
		writeJson(writer, objects.Results)
	})

	// 删除文件
	mux.HandleFunc("DELETE /api/v1/repos/{id}/branches/main/files/{filename}", func(writer http.ResponseWriter, request *http.Request) {
		id := request.PathValue("id")
		filename := request.PathValue("filename")
		ctx := context.Background()
		err := lakefsClient.Objects.DeleteObject(ctx, id, "main", &lakefs.DeleteObjectOptions{
			Path: filename,
		})
		if err != nil {
			writeError(writer, err)
			return
		}
		writeJson(writer, PostFilesResponse{
			Total: 1,
		})
	})

	// 查看文件
	mux.HandleFunc("GET /api/v1/repos/{id}/branches/main/files/{filename}", func(writer http.ResponseWriter, request *http.Request) {
		id := request.PathValue("id")
		filename := request.PathValue("filename")
		preview := request.URL.Query().Get("preview") == "true"
		ctx := context.Background()
		opts := &lakefs.GetObjectContentOptions{
			Path: filename,
		}
		if preview {
			opts.Range = &lakefs.RangeByteSize{
				End: 50 * 1024, // 50KB
			}
		}
		file, err := lakefsClient.Objects.GetObjectContent(ctx, id, "main", opts)
		if err != nil {
			writeError(writer, err)
			return
		}
		writer.Header().Set("Content-Type", "application/octet-stream")
		_, err = io.Copy(writer, file.Body)
		if err != nil {
			writeError(writer, err)
			return
		}
	})

}

type PostFilesRequest struct {
	Files []*lakefs.StagingMetadata `json:"files"`
}

type PostFilesResponse struct {
	Total int `json:"total"`
}
