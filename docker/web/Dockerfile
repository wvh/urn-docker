FROM golang:1 AS build

ENV GO111MODULE=on
WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download -x

COPY . .
#RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -o=/bin/ ./cmd/...
RUN GO111MODULE=on CGO_ENABLED=0 make all

FROM scratch
COPY --from=build /app/bin/ /bin/
EXPOSE 8080
ENTRYPOINT ["/bin/web"]
