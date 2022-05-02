FROM debian:stable-20220328-slim
LABEL maintainer=panfeng
EXPOSE 12345
WORKDIR /root
RUN mkdir -p /root/.kube
COPY kube-scheduler-main /root
CMD "./kube-scheduler-main"
