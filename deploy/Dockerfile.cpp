FROM debian:bullseye-slim

RUN apt-get update && apt-get install -y \
    build-essential \
    cmake \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY src/ .

RUN mkdir -p build && cd build && \
    cmake .. && make && \
    cp alg /app/alg && \
    cd .. && rm -rf build

RUN apt-get purge -y build-essential cmake && \
    apt-get autoremove -y && \
    rm -rf /var/lib/apt/lists/*

ENTRYPOINT ["/usr/local/bin/runner"]
