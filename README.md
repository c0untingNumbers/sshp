# sshp
`sshp` is a command-line program to simplify the management of SSH keys stored in the 1Password agent configuration file. It uses the Bubbletea Text User Interface framework to provide an interface to toggle the SSH keys within the configuration, allowing users to quikcly select or deselect keys for use.

## Purpose
When you have too many SSH keys in your 1Password, you may encounter a problem where you SSH into a server and you get a disconnection from the server due to `Too many authentication failures` error. `sshp` allows you to manage which keys are active or not easily.

I made this just so I can get more experience in Golang as well as using a TUI framework.

## Installation
Clone the repository and build the program:

```bash
$ git clone https://github.com/c0untingNumbers/sshp
$ cd sshp
$ go build -o sshp
```
Run the program:
```bash
$ ./sshp
```

## Requirements
- Go programming language
- 1Password SSH agent configured with agent.toml

## Contribution
Contributions are welcome! Feel free to fork the repository and submit pull requests