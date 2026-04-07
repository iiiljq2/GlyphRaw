FROM --platform=linux/amd64 pytorch/pytorch:2.0.0-cuda11.7-cudnn8-runtime

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    libgl1 \
    libglib2.0-0 \
    procps \
    wget \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

RUN git clone --depth 1 https://github.com/yeungchenwa/FontDiffuser.git .
COPY pip_packages /tmp/pip_packages
COPY . .

RUN mkdir -p /app/ckpt /app/data_examples

RUN python -m pip install --no-cache-dir --no-index --find-links=/tmp/pip_packages \
    huggingface_hub==0.19.4 \
    transformers==4.33.1 \
    accelerate==0.23.0 \
    diffusers==0.22.0 \
    gradio==4.8.0 \
    pyyaml \
    pygame \
    opencv-python \
    info-nce-pytorch \
    kornia

RUN rm -rf /root/.cache/pip /tmp/pip_packages

VOLUME ["/app/ckpt", "/app/data_examples", "/output_data"]

CMD ["python", "gradio_app.py"]