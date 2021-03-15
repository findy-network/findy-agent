aws sts get-caller-identity | grep Account | sed 's/^[ \t]*"Account": "\(.*\)",/\1/'
