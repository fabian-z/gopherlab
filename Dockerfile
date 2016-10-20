# docker build -t gopherlab .
# docker run --net host -v $HOME:/var/user gopherlab
# open http://localhost:8888 in your browser

FROM pritunl/archlinux:latest

RUN echo 'Server = https://mirrors.kernel.org/archlinux/$repo/os/$arch' > /etc/pacman.d/mirrorlist
RUN pacman -Syyu --noconfirm
RUN pacman -S --noconfirm --needed go go-tools zeromq jupyter jupyter-notebook git base-devel mathjax pandoc texlive-core wget
RUN pacman -S --needed --noconfirm ipython python-ipykernel python-setuptools python-jinja python-pyzmq python-jsonschema python-mistune python-pygments python-setuptools python2-setuptools npm jupyter-nbconvert qt5-svg python-pyqt5 python-sip
RUN usermod -d /tmp/ nobody

USER nobody

RUN cd /tmp && git clone https://aur.archlinux.org/jupyterlab-git.git
RUN cd /tmp/jupyterlab-git/ && makepkg --noconfirm

USER root
RUN cd /tmp/jupyterlab-git && pacman -U --noconfirm *.pkg.tar.xz
RUN mkdir /var/jupyter && \
    useradd -M -U -d /var/jupyter jupyter && \
    mkdir /var/user && \
    chown jupyter -R /var/user && \
    chown jupyter -R /var/jupyter
USER jupyter

RUN mkdir /tmp/go
RUN GOPATH=/tmp/go \
    go get github.com/fabian-z/gopherlab
RUN mkdir -p ~/.local/share/jupyter/kernels/gopherlab
RUN cp -a /tmp/go/src/github.com/fabian-z/gopherlab/kernel/* ~/.local/share/jupyter/kernels/gopherlab/
RUN cp -a /tmp/go/bin/gopherlab ~/.local/share/jupyter/kernels/gopherlab/
RUN sed -i "s#/go/bin/gopherlab#$HOME/.local/share/jupyter/kernels/gopherlab/gopherlab#g" $HOME/.local/share/jupyter/kernels/gopherlab/kernel.json

RUN jupyter serverextension enable --py jupyterlab

WORKDIR /var/user

EXPOSE 8888
CMD ["jupyter", "lab"]

#For classical notebook use:
#CMD ["jupyter", "notebook"]
