package main

import (
	"log"
	"os"
	"os/exec"
)

var cmdChain = []*exec.Cmd{
	exec.Command("lib/synonyms"),
	exec.Command("lib/sprinkle"),
	exec.Command("lib/coolify"),
	exec.Command("lib/domainify"),
	exec.Command("lib/available"),
}

func main() {
	//	データは標準入力から読み込まれ、処理結果は標準出力に書き出される
	//	一つ目のプログラムつまりsynonymsによっての標準入力のストリーム（Stdin）を、domainfinderにとって標準入力（os.Stdin）に接続している。
	cmdChain[0].Stdin = os.Stdin
	//	最後のプログラムつまりavailableにとっての標準出力のストリーム（Stdout）を、domainfinderにとっての標準出力（os.Stdout）に接続している
	cmdChain[len(cmdChain)-1].Stdout = os.Stdout

	for i := 0; i < len(cmdChain)-1; i++ {
		thisCmd := cmdChain[i]
		nextCmd := cmdChain[i+1]
		stdout, err := thisCmd.StdoutPipe()
		if err != nil {
			log.Panicln(err)
		}
		nextCmd.Stdin = stdout
	}

	for _, cmd := range cmdChain {
		if err := cmd.Start(); err != nil {
			log.Panicln(err)
		} else {
			defer cmd.Process.Kill()
		}
	}

	for _, cmd := range cmdChain {
		if err := cmd.Wait(); err != nil {
			log.Panicln(err)
		}
	}
}
