FROM scratch
LABEL maintainer="https://github.com/gzlj"
COPY thanos-reloader /bin/
entrypoint [ "/bin/thanos-reloader" ]
