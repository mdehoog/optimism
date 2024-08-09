FROM golang:1.22

RUN apt-get update && apt-get install -y jq

ENV REPO=https://github.com/mdehoog/op-geth.git
ENV VERSION=witness
ENV COMMIT=0d005939aa2f2ccb1d4f3d3b65d2507110c6cb5b
# avoid depth=1, so the geth build can read tags
RUN mkdir op-geth && cd op-geth && \
    git clone $REPO --branch $VERSION --single-branch . && \
    git switch -c branch-$VERSION $COMMIT && \
    bash -c '[ "$(git rev-parse HEAD)" = "$COMMIT" ]' || \
    (echo "Dockerfile COMMIT is not equal to repo HEAD" && exit 1)
RUN cd op-geth && go run build/ci.go install -static ./cmd/geth
RUN cp ./op-geth/build/bin/geth /usr/local/bin

COPY l2-op-geth-entrypoint.sh /entrypoint.sh

VOLUME ["/db"]

ENTRYPOINT ["/bin/sh", "/entrypoint.sh"]
