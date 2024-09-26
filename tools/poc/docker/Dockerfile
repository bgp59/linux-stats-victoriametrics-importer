FROM ubuntu:20.04

RUN \
    apt update \
    && ln -fs /usr/share/zoneinfo/GMT /etc/localtime \
        && apt-get install -y tzdata \
    && (yes | unminimize) \
    && apt-get install -y \
        curl \
        gcc \
        libcap2-bin \
        python3.8-full \
        python3-pip \
        man-db \
        rsync \
        strace \
    && pip install -U pip

ENV TINI_VERSION=v0.19.0
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini /tini
RUN chmod +x /tini
ENTRYPOINT ["/tini", "--"]

RUN useradd -U -d /home/ubuntu -m -s /bin/bash ubuntu

WORKDIR /usr/local/lsvmi-poc


COPY files/base/ .
RUN \
    victoria-metrics/download-victoria-metrics.sh \
    && victoria-metrics/create-victoria-metrics-runtime-symlinks.sh
RUN \
    grafana/download-grafana.sh \
    && grafana/create-grafana-runtime-symlinks.sh

COPY files/update/ .

USER ubuntu
CMD ["./run-poc.sh"]
