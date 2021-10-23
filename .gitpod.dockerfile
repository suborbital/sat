FROM gitpod/workspace-full
USER gitpod

RUN brew tap suborbital/subo && \
    brew install subo
