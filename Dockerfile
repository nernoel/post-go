FROM golang:1.22.3-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/postgo .

FROM alpine:3.20

RUN adduser -D -H postgo
USER postgo
WORKDIR /home/postgo

COPY --from=build /bin/postgo /usr/local/bin/postgo

ENTRYPOINT ["postgo"]
