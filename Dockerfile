FROM --platform=linux/amd64 pytorch/pytorch:1.13.1-cuda11.6-cudnn8-runtime

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update --fix-missing && apt-get install -y --no-install-recommends \
    software-properties-common \
    && add-apt-repository ppa:deadsnakes/ppa -y \
    && apt-get update && apt-get install -y --no-install-recommends \
    python3.9 \
    python3.9-distutils \
    python3.9-dev \
    git \
    libgl1 \
    libglib2.0-0 \
    procps \
    wget \
    curl \
    && rm -rf /var/lib/apt/lists/*

RUN update-alternatives --install /usr/bin/python python /usr/bin/python3.9 10 \
    && update-alternatives --install /usr/bin/python3 python3 /usr/bin/python3.9 10 \
    && curl -sS https://bootstrap.pypa.io/get-pip.py | python3.9

WORKDIR /app


COPY pip_packages /tmp/pip_packages
COPY . .

RUN mkdir -p /app/ckpt /app/data_examples

RUN python -m pip install --no-cache-dir --no-index --find-links=/tmp/pip_packages /tmp/pip_packages/*.whl

RUN rm -rf /root/.cache/pip /tmp/pip_packages

VOLUME ["/app/ckpt", "/app/data_examples", "/output_data"]

CMD ["python", "gradio_app.py"]