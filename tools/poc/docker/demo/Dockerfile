FROM lsvmi-victoria-metrics-grafana-ubuntu_20.04

COPY tools/poc/files/lsvmi/*lsvmi* lsvmi/
COPY bin lsvmi/bin
USER root
RUN \
    ln -fs /volumes/runtime/lsvmi/out lsvmi \
    && ln -fs /volumes/runtime/lsvmi/log lsvmi
USER ${POC_USER}
