FROM golang:1.24.4 AS build-stage

WORKDIR /app
# copy all, because some code is in subdirs and only the binary will be copied to the run stage
COPY . .
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /oauth-static-webserver

FROM alpine:latest AS run

COPY --from=build-stage /oauth-static-webserver /oauth-static-webserver

EXPOSE 8080

CMD ["/oauth-static-webserver"]
