# Demo tape for bili. Rendered with ascii-gif (github.com/tamnd/ascii-gif),
# which supplies the window chrome and theme; this file is just the action.
#
#   ascii-gif render docs/demo/bili.tape -o docs/static/demo.gif
#
# bili must be on PATH inside the recording shell.

Hide
Type "export PS1='$ ' PATH=/Users/apple/bin:$PATH"
Enter
Type "clear"
Enter
Show

Sleep 500ms
Type "bili search 'lofi' -n 5 --fields bvid,title,owner_name,view_count"
Sleep 800ms
Enter
Sleep 3s

Type "bili video BV17x411w7KC --fields bvid,title,owner_name,view_count,like_count"
Sleep 800ms
Enter
Sleep 2.5s

Type "bili search 'lofi' -n 4 -o jsonl | jq -r .title"
Sleep 800ms
Enter
Sleep 3s
