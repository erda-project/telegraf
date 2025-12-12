FROM registry.erda.cloud/erda-x/golang:1.17-buster-archive AS build
 
RUN apt-get update && apt-get -y install libpcap-dev

COPY . /root/build
WORKDIR /root/build

RUN make telegraf

FROM registry.erda.cloud/erda-x/oraclelinux:7

WORKDIR /app
RUN mkdir -p /app/conf && yum -y install sysstat ntp libpcap http://mirror.centos.org/$(if [[ $(uname -m) == x86_64 ]]; then echo centos; else echo altarch; fi)/7/updates/$(uname -m)/Packages/libpcap-devel-1.5.3-13.el7_9.$(uname -m).rpm

COPY --from=build /root/build/telegraf /app/
COPY --from=build /root/build/conf /app/conf
COPY --from=build /root/build/exec_scripts /app/exec_scripts
COPY --from=build /root/build/entrypoint.sh /app/

CMD ["./entrypoint.sh"]
