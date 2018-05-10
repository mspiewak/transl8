FROM golang:alpine AS build-env

WORKDIR /go/src/app
COPY . .

RUN apk add --no-cache curl git
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
RUN dep ensure
RUN go build -o app

FROM alpine
WORKDIR /app
COPY --from=build-env /go/src/app/app /app/
ENTRYPOINT ./app