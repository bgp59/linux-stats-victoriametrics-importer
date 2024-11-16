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
        sudo \
        strace \
    && pip install -U pip

ENV TINI_VERSION=v0.19.0
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini /tini
RUN chmod +x /tini
ENTRYPOINT ["/tini", "--"]

RUN \
    echo "ubuntu ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/ubuntu && \
    chmod 0440 /etc/sudoers.d/ubuntu


ENV POC_USER=ubuntu

RUN useradd -U -d /home/${POC_USER} -m -s /bin/bash ${POC_USER}
RUN \
    mkdir -p /volumes/runtime \
    && chown ${POC_USER}:${POC_USER} /volumes/runtime

VOLUME /volumes/runtime

WORKDIR /home/${POC_USER}
USER ${POC_USER}
CMD ["sleep", "infinity"]
