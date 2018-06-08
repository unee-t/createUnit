FROM scratch
COPY unit /
WORKDIR /
COPY sql/ .
ENV PORT 9000
ENTRYPOINT ["/unit"]
