FROM golang:1.22 AS build-stage
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /cashier-status

FROM gcr.io/distroless/static-debian11 as build-release-stage

WORKDIR /

COPY --from=build-stage /cashier-status /cashier-status
#USER nonroot:nonroot

EXPOSE 8888
CMD ["/gin-helloworld"]