ARG FROMIMAGESUFFIX
FROM lsvmi-victoria-metrics-grafana-ubuntu-20.04${FROMIMAGESUFFIX}
ARG TARGETPLATFORM

ENV _os_arch=${TARGETPLATFORM//\//-}
COPY tools/poc/files/lsvmi/*lsvmi* lsvmi/
COPY bin/linux-stats-*${_os_arch}* lsvmi/bin/
COPY bin/${_os_arch} lsvmi/bin/${_os_arch}/
USER root
RUN \
    ln -fs /volumes/runtime/lsvmi/out lsvmi \
    && ln -fs /volumes/runtime/lsvmi/log lsvmi
USER ${POC_USER}
