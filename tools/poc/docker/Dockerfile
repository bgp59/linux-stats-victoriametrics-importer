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

COPY files /usr/local/lsvmi-poc
WORKDIR /usr/local/lsvmi-poc
RUN victoria-metrics/download-victoria-metrics.sh \
    && victoria-metrics/create-victoria-metrics-runtime-symlinks.sh
RUN grafana/download-grafana.sh \
    && grafana/create-grafana-runtime-symlinks.sh
RUN ln -fs /volumes/linux-stats-victoriametrics-importer/tools/poc/lsvmi .

RUN useradd -U -d /home/ubuntu -m -s /bin/bash ubuntu
USER ubuntu
CMD ["sleep", "infinity"]

