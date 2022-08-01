FROM uhub.service.ucloud.cn/entropypool/alpine:3.16

WORKDIR /app

COPY ledger-gateway /app

ENTRYPOINT ["/app/ledger-gateway"]

EXPOSE 50410 50411 50412 50413

CMD ["run"]
