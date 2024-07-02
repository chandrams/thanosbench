FROM quay.io/prometheus/busybox:latest

COPY .build/thanosbench /bin/thanosbench

RUN adduser \
    -D `#Dont assign a password` \
    -H `#Dont create home directory` \
    -u 1001 `#User id`\
    thanosbench && \
    chown thanosbench /bin/thanosbench
USER 1001

ENTRYPOINT [ "/bin/thanosbench" ]
