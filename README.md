GopherJS - A transpiler from Go to JavaScript
---------------------------------------------

### Useful Links
- Try GopherJS on the [GopherJS Playground](http://neelance.github.io/gopherjs-playground/)
- Get help in the [Google Group](https://groups.google.com/d/forum/gopherjs)

### What is GopherJS?
GopherJS translates [Go code](http://golang.org/) to pure JavaScript code. Its main purpose is to give you the opportunity to write front-end code in Go which will still run in all browsers.

You can take advantage of Go's elegant type system and other compile-time checks that can have a huge impact on bug detection and the ability to refactor, especially for big projects. Just think of how often a JavaScript method has extra handling of some legacy parameter scheme, because you don't know exactly if some other code is still calling it in that old way or not. GopherJS will tell you and if it does not complain, you can be sure that this kind of bug is not present any more.

### What is supported?
- all basic types, including 64-bit integers and complex numbers
- arrays, slices, maps and structures
- full type system with support for interfaces and type assertions
- reflection for all types
- many packages have been successfully tested, see [compatibility table](doc/packages.md)
- the transpiler can turn itself into pure JavaScript that runs in all major browsers

### Installation and Usage
Get GopherJS and dependencies with: 
```
go get github.com/neelance/gopherjs
```
Now you can use  `./bin/gopherjs build` and `./bin/gopherjs install` which behave similar to the `go` tool. The generated JavaScript files can be used as usual in a website. Go's `println` builtin prints to the JavaScript console via `console.log`.

*Note: GopherJS will try to write compiled object files of the core packages to your $GOROOT/pkg directory. If that fails, it will fall back to $GOPATH/pkg.*

### Node.js
You can also run the generated code with Node.js instead of a browser. However, system calls (e.g. writing to the console via the `fmt` package or most of the `os` functions) will not work until you compile and install the syscall module. If you just need console output, you can use `println` instead.
The syscall module currently only supports OS X. Please tell me if you would like to have support for other operating systems. On OS X, get the latest Node.js 0.11 release from [here](http://blog.nodejs.org/release/) or via `brew install node --devel`. Then compile and install the module:
```
npm install --global node-gyp
cd src/github.com/neelance/gopherjs/node-syscall/
node-gyp rebuild
mkdir -p ~/.node_libraries/
cp build/Release/syscall.node ~/.node_libraries/syscall.node
cd ../../../../../
```

### Roadmap
These features are not implemented yet, but on the roadmap:

- optimizations of generated JavaScript code for improved output size and performance
- goroutines, channels, select
- goto
- output minification
- source maps

### Deviations from Go specification
Some tradeoffs had to be made in order to avoid huge performance impacts. Please get in contact if those are deal breakers for you.

- int and uint do not overflow, but have a maximum precision of 53 bits, after which rounding errors might occur (all other integer types have simulated overflow)
- float32 and complex64 have the same precision as float64 and complex128

### Interface to native JavaScript
A function's body can be written in JavaScript by putting the code in a string constant with the name `js_[function name]` for package functions and `js_[type name]_[method name]` for methods. In that case, GopherJS disregards the Go function body and instead generates `function(...) { [constant's value] }`. This allows functions to have a Go signature that the type checker can use while being able to call native JavaScript functions.
When calling such a native function from Go, the arguments and return values are converted between Go types and JavaScript types according to the table below. The second column denotes the types that are used when converting to `interface{}`. Multiple values can be returned by using an array.

| Go types            | Go interface type      | JavaScript type |
| ------------------- | ---------------------- | --------------- |
| bool                | bool                   | Boolean         |
| int*, uint*, float* | float64                | Number          |
| string              | string                 | String          |
| slices, arrays      | []interface{}          | Array           |
| maps                | map[string]interface{} | Object          |

Types not listed are passed through. Please note that this interface is subject to change. Suggestions are always welcome.

### Libraries
[go-angularjs](https://github.com/neelance/go-angularjs) - a wrapper for [AngularJS](http://angularjs.org)

[![Analytics](https://ga-beacon.appspot.com/UA-46799660-1/gopherjs/README.md)](https://github.com/igrigorik/ga-beacon)
