# 最小镜像：静态二进制直接跑在 scratch 上，仅构建阶段依赖 Go 基础镜像
FROM golang:1.22-bookworm AS builder

WORKDIR /src

# 依赖随仓库 vendor 提交，构建无需联网拉取模块
COPY . .

ENV CGO_ENABLED=0 GOOS=linux
RUN go build -mod=vendor -trimpath -ldflags="-s -w" -o /out/gravity-server ./cmd/server

FROM scratch
COPY --from=builder /out/gravity-server /gravity-server

EXPOSE 8080
USER 65534:65534
ENTRYPOINT ["/gravity-server"]
