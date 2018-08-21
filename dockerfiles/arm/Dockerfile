FROM arm32v6/alpine:3.6
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
COPY n2bot /
WORKDIR /

ENTRYPOINT ["/n2bot"]