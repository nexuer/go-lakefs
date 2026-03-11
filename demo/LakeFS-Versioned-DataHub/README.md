# LakeFS-Versioned-DataHub

数据仓库demo示例，包含前端和后端
- 前端: vue + naive-ui
- 后端: go http + go-lakefs

## 演示
### 启动前端
```shell
sh -c "cd web && pnpm i && pnpm dev"
```
### 启动后端
```shell
export LAKEFS_ENDPOINT=127.0.0.1:38000
export LAKEFS_ACCESS_KEY_ID=your_access_key
export LAKEFS_SECRET_ACCESS_KEY=your_access_secret
go run ./main.go
```
