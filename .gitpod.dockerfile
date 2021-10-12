FROM gitpod/workspace-full
USER gitpod

RUN curl -sL https://raw.githubusercontent.com/moovweb/gvm/master/binscripts/gvm-installer| bash

RUN ["/bin/bash", "-c", ". /home/gitpod/.gvm/scripts/gvm && gvm install go1.17 -B"]
RUN ["/bin/bash", "-c", ". /home/gitpod/.gvm/scripts/gvm && gvm use go1.17"]

    

