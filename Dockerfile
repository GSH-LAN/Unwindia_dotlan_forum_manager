FROM golang:1.19.4-alpine AS build-env
ADD . /app
WORKDIR /app
ARG TARGETOS
ARG TARGETARCH
RUN go mod download
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o unwindia_dotlan_forum_manager ./cmd/unwindia_dotlan_forum_manager

# Runtime image
FROM redhat/ubi8-minimal:8.7

RUN rpm -ivh https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm
RUN microdnf update && microdnf -y install ca-certificates inotify-tools

COPY --from=build-env /app/unwindia_dotlan_forum_manager /
EXPOSE 8080
CMD ["./unwindia_ms_dotlan"]
