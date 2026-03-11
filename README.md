# go-lakefs

[![Go](https://img.shields.io/badge/Go-1.20+-blue.svg)](https://golang.org/)

## 项目简介

**go-lakefs** 是 lakeFS 的 Golang SDK，旨在提供类型安全、简洁易用的接口，让 Go 开发者能够方便地与 lakeFS 服务器交互。

lakeFS 是一个开源的数据版本控制系统，支持 Git 风格的分支、提交、合并、标签等操作，非常适合数据湖、数据管道、机器学习模型版本管理等场景。

该 SDK 基于 lakeFS 官方 REST API（v1）实现，参考文档：  
https://docs.lakefs.io/latest/reference/api/

## 特性

- 类型安全的结构体封装（Repository、Branch、Commit、ObjectStats 等）
- 支持核心仓库、分支、提交、对象操作
- 流式上传/下载对象，支持大文件（待完善）
- 使用 context.Context 控制超时与取消
- 仅依赖少量标准库和必要第三方包

## 安装

```shell
go get github.com/nexuer/go-lakefs
```

## 本地测试环境
仓库提供了一个简单的单机 lakeFS 部署配置，使用 PostgreSQL + Minio存储。
- [docker-compose.yaml](./docker-compose.yaml)
### 启动
```shell
docker compose up -d
```

1. 初次访问：http://localhost:38000
2. 按照页面提示设置管理员账号和密钥。

#### 停止 & 卸载
- 停止但保留数据：
```shell
docker compose down
```
- 彻底删除（包括数据）：
```shell
docker compose down --volumes
```

## 相关链接
- lakeFS 官方文档：https://docs.lakefs.io/
- lakeFS REST API 参考：https://docs.lakefs.io/latest/reference/api/
- lakeFS GitHub：https://github.com/treeverse/lakeFS