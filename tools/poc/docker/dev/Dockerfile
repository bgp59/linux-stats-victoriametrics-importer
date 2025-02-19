FROM lsvmi-base

USER root
WORKDIR /usr/local/lsvmi-poc

COPY files/base .
RUN \
    victoria-metrics/download-victoria-metrics.sh \
    && victoria-metrics/create-victoria-metrics-runtime-symlinks.sh
RUN \
    grafana/download-grafana.sh \
    && grafana/create-grafana-runtime-symlinks.sh

COPY files/update/ .

USER ${POC_USER}
CMD ["./run-poc.sh"]
