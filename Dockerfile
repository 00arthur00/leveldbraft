FROM alpine:latest


LABEL Name=leveldbraft

RUN apk --no-cache add ca-certificates wget curl

COPY leveldbraft /usr/local/bin/leveldbraft

ENTRYPOINT ["/usr/local/bin/leveldbraft"]

EXPOSE 8901 8902