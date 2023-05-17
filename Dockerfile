FROM alpine:3.15

ADD ./src/app /bin/app

ENTRYPOINT [ "app" ]