FROM registry.erda.cloud/erda-x/all-in-one:7 as build

RUN yum -y install sysstat ntp libpcap libpcap-devel

COPY . /root/build
WORKDIR /root/build

RUN make telegraf

FROM registry.erda.cloud/erda-x/oraclelinux:7

WORKDIR /app
RUN mkdir -p /app/conf && yum -y install sysstat ntp libpcap libpcap-devel

COPY --from=build /root/build/telegraf /app/
COPY --from=build /root/build/conf /app/conf
COPY --from=build /root/build/exec_scripts /app/exec_scripts
COPY --from=build /root/build/entrypoint.sh /app/

CMD ["./entrypoint.sh"]
