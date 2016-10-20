# gopherlab - Go in Jupyter Notebooks
[![CI status](https://travis-ci.org/fabian-z/gopherlab.svg)](https://travis-ci.org/fabian-z/gopherlab)
[![Code quality report](https://goreportcard.com/badge/github.com/fabian-z/gopherlab)](https://goreportcard.com/report/github.com/fabian-z/gopherlab)

`gopherlab` is a Go kernel for [Jupyter](http://jupyter.org/) notebooks, also supporting the still alpha [JupyterLab](https://github.com/jupyter/jupyterlab).
This allows for using Go in an interactive context, directly in the browser, making it ideal for educational uses and data science. 
Use `gopherlab` to create and share documents that contain live Go code, equations, visualizations and explanatory text.  These notebooks can then be shared with others via e-mail, GitHub, etc. 

The original `gophernotes` project came out of the [Gopher Gala](http://gophergala.com/) 2016 and was inspired by a self-described as limited ipython kernel called [iGo](https://github.com/takluyver/igo).
The REPL backend of both `gophernotes` and `gopherlab` is provided by [gore](https://github.com/motemen/gore).

## Screenshots/Examples

### Simple interactive use:

![Screenshot](https://rawgit.com/fabian-z/gopherlab/master/doc/screenshot.png)

### Example Notebooks (download and run them locally, follow the links to view in GitHub):
- [Simple Printing and Channels](https://github.com/fabian-z/gopherlab/blob/master/examples/Simple-Example.ipynb)
- [Pattern Recognition with Golearn](https://github.com/fabian-z/gopherlab/blob/master/examples/Pattern-Recognition.ipynb)
- [Feed Forward, Recurrent Neural Nets](https://github.com/fabian-z/gopherlab/blob/master/examples/Feed-Forward-Recurrent-NN.ipynb)
- [Time Parsing, Formatting](https://github.com/fabian-z/gopherlab/blob/master/examples/Time-Formatting-Parsing.ipynb)
- [Stateful Goroutines](https://github.com/fabian-z/gopherlab/blob/master/examples/Stateful-Goroutines.ipynb)
- [Worker Pools](https://github.com/fabian-z/gopherlab/blob/master/examples/Worker-Pools.ipynb)

## Installation

### Local, Linux

- Dependencies:

  - [Go](https://golang.org/) (Tested with Go 1.5 and 1.6)
  - Jupyter (see [here](http://jupyter.readthedocs.org/en/latest/install.html) for more details on installing jupyter)
  - [ZeroMQ](http://zeromq.org/) (4.x, if you need another version please file an issue)

- Create a workspace and setup your `GOPATH`, see https://golang.org/doc/code.html#GOPATH



-    ```
    go get github.com/fabian-z/gopherlab
    ```
  

- Create a directory for the new kernel config:

  ```
  mkdir -p ~/.local/share/jupyter/kernels/gopherlab
  ```

- Copy the kernel config into the `jupyter` directory:

  ```
  cp -r $GOPATH/src/github.com/fabian-z/gopherlab/kernel/* ~/.local/share/jupyter/kernels/gopherlab/
  ```
  
  Note, depending on which version of jupyter you are using and if you are using Anaconda, you may need to copy to `~/.ipython` rather than `~/.local/share`.

- Copy the gopherlab kernel to your kernels directory and add the correct path in `kernel.json`:

```
  cp -a $GOPATH/bin/gopherlab ~/.local/share/jupyter/kernels/gopherlab/
  
  sed -i "s#/go/bin/gopherlab#$HOME/.local/share/jupyter/kernels/gopherlab/gopherlab#g" $HOME/.local/share/jupyter/kernels/gopherlab/kernel.json
```
  


### Local, OSX

TBD

## Getting Started

- If you completed the install above start the jupyter notebook:

  ```
  jupyter notebook
  ```

- Alternatively, start the JupyterLab Alpha
 ```
 jupyter lab
 ```

- Select `Go (gopherlab)` from the `New` drop down menu.

- Have Fun!


## Troubleshooting

### gopherlab not found
- You will need to change the path to the `gopherlab` executable in `kernel/kernel.json`.  Above docs provide a `sed` command for this purpose. You should put the **full path** to the `gopherlab` executable here, and shouldn't have any further issues.


## Custom Commands
Some of the custom commands from the [gore](https://github.com/motemen/gore) REPL have carried over to `gopherlab`.  Note, in particular, the syntax for importing packages:

```
:import <package path>  Import package
:print                  Show current source (currently prints to the terminal where the notebook server is running)
:write [<filename>]     Write out current source to file
:help                   List commands
```

Output support for these command is currently under construction, e.g. `:print` already works.

## Licenses

Original `gophernotes` was created by [Daniel Whitenack](http://www.datadan.io/). `gopherlab` was forked by Fabian Zaremba, in order to add new features, support new message spec (JupyterLab) and update several core components. Both projects are licensed under an [MIT-style License](LICENSE.md).

