FROM archlinux:base-devel

WORKDIR /app

COPY cmd/cmd /app/cmd/
COPY config/config.yaml /app/config/

ENTRYPOINT [ "./cmd/cmd" ]