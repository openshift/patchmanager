FROM registry.centos.org/centos:8
ADD ./patchmanager /usr/bin/patchmanager
EXPOSE 8080
CMD /usr/bin/patchmanager