FROM scratch
ENV PATH=/bin

COPY echo /bin/

WORKDIR /

ENTRYPOINT ["/bin/echo"]