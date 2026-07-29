# Demo GIF source. Regenerate from the repo root with ctxlog on PATH:
#   vhs assets/demo.tape
# Requires vhs (https://github.com/charmbracelet/vhs): brew install vhs

Output assets/demo.gif

Set FontSize 15
Set Width 1100
Set Height 420
Set Padding 16
Set TypingSpeed 25ms
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful

Hide
Type "cd $(mktemp -d) && clear"
Enter
Show

Type "# an agent logs progress as it works"
Enter
Sleep 300ms
Type `ctxlog append -shard="payments" -msg="extracted PaymentGateway interface, tests green" -agent="claude-main"`
Enter
Sleep 1s

Type "# 3 subagents append to the SAME journal in parallel"
Enter
Sleep 300ms
Type `( for i in 1 2 3; do ctxlog append -shard="payments" -msg="module $i migrated" -agent="sub-$i" & done; wait )`
Enter
Sleep 1.5s

Type "# new session: catch up instantly"
Enter
Sleep 300ms
Type `ctxlog read -shard="payments"`
Enter
Sleep 4s
