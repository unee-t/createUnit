FROM scratch
COPY unit /
ENV PORT 9000
ENTRYPOINT ["/unit"]
