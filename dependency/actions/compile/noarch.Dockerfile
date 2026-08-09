FROM ubuntu:noble
# Copyright (c) 2013-Present CloudFoundry.org Foundation, Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0

ENV DEBIAN_FRONTEND=noninteractive
ENV LC_CTYPE='en_US.UTF'

RUN apt-get update && apt install python3-pip -y

COPY entrypoint /entrypoint
COPY constraints.py /constraints.py

ENTRYPOINT ["/entrypoint"]
