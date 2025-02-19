FROM lsvmi-dev
ARG TARGETPLATFORM

COPY staging/${TARGETPLATFORM}/lsvmi lsvmi/
USER root
RUN \
    ln -fs /volumes/runtime/lsvmi/out lsvmi \
    && ln -fs /volumes/runtime/lsvmi/log lsvmi
USER ${POC_USER}
