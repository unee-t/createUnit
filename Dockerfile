FROM scratch
COPY unit /
WORKDIR /
COPY sql/ /sql/
ENV PORT 9000
ENTRYPOINT ["/unit"]
