FROM golang:alpine AS build-env

ARG G_API_KEY
ENV ENV_G_API_KEY=$G_API_KEY

WORKDIR /go/src/app
COPY . .

RUN apk add --no-cache curl git
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
RUN dep ensure
RUN go build -o app

FROM alpine
RUN echo $ENV_G_API_KEY
WORKDIR /app
COPY --from=build-env /go/src/app/app /app/
ENTRYPOINT ./app --gApiKey $ENV_G_API_KEY