package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ktr0731/go-fuzzyfinder"
)

var (
	tf string
)

func main() {
	tf = os.Getenv("TFMIG_TF_PATH")
	if tf == "" {
		tf = "terraform"
	}
	if err := exec.Command(tf, "version").Run(); err != nil {
		log.Fatal(err)
	}

	srcProj := flag.String("s", "", "source location path of the terraform project")
	dstProj := flag.String("d", "", "destination location path of the terrafrom project")
	workspace := flag.String("w", "", "terraform workspace name")

	flag.Parse()

	if *srcProj == "" {
		log.Fatal("source location path (-s) is must be specified to use tfmig")
	}

	if *dstProj == "" {
		log.Fatal("destination location path (-d) is must be specified to use tfmig")
	}

	tmpDir := "/tmp/tfmig"
	srcBakDir := fmt.Sprintf("%s/.bak", *srcProj)
	dstBakDir := fmt.Sprintf("%s/.bak", *dstProj)
	for _, bak := range []string{srcBakDir, dstBakDir, tmpDir} {
		if _, err := os.Stat(bak); os.IsNotExist(err) {
			if err := os.Mkdir(bak, 0755); err != nil {
				log.Fatal(err)
			}
		}
	}
	defer os.RemoveAll(tmpDir)

	if *workspace != "" {
		out, err := terraform(*srcProj, "workspace", "select", *workspace)
		if err != nil {
			log.Fatalf("%s\n%v", out, err)
		}
		fmt.Fprintln(os.Stdout, out)
	}

	selectedStates, err := selectStates(*srcProj)
	if err != nil {
		log.Fatal(err)
	}

	srcState := "/tmp/tfmig/src.tfstate"
	dstState := "/tmp/tfmig/dst.tfstate"

	day := time.Now().Format("2006-01-02-15-04-05")
	for _, dat := range []struct {
		Proj          string
		StateFilename string
		BakDirname    string
	}{
		{*srcProj, srcState, srcBakDir},
		{*dstProj, dstState, dstBakDir},
	} {
		out, err := terraform(dat.Proj, "state", "pull")
		if err != nil {
			log.Fatalf("%s\n%v", out, err)
		}
		for _, name := range []string{
			fmt.Sprintf("%s/%s.tfstate", dat.BakDirname, day),
			dat.StateFilename,
		} {
			if err := os.WriteFile(name, []byte(out), 0755); err != nil {
				log.Fatal(err)
			}
		}
	}

	for _, st := range selectedStates {
		out, err := terraform(*dstProj, "state", "mv", fmt.Sprintf("-state=%s", srcState), fmt.Sprintf("-state-out=%s", dstState), st, st)
		if err != nil {
			log.Fatalf("%s\n%v", out, err)
		}
		fmt.Fprintln(os.Stdout, out)
	}

	for _, dat := range []struct {
		Proj            string
		StateFilename string
	}{
		{*dstProj, dstState},
		{*srcProj, srcState},
	} {
		out, err := terraform(dat.Proj, "state", "push", dat.StateFilename)
		if err != nil {
			log.Fatalf("%s\n%v", out, err)
		}
		fmt.Fprintln(os.Stdout, out)
	}

	for _, proj := range []string{*dstProj, *srcProj} {
		out, err := terraform(proj, "state", "list")
		if err != nil {
			log.Fatalf("%s\n%v", out, err)
		}
		fmt.Fprintln(os.Stdout, out)
	}
}

func terraform(project string, args ...string) (string, error) {
	log.Println(tf, fmt.Sprintf("-chdir=%s", project), args)
	cmd := exec.Command(tf, append([]string{fmt.Sprintf("-chdir=%s", project)}, args...)...)

	cmdStdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stdoutScanner := bufio.NewScanner(cmdStdout)

	cmdStderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}
	stderrScanner := bufio.NewScanner(cmdStderr)

	bufch := make(chan string)
	go func() {
		for _, s := range []*bufio.Scanner{stdoutScanner, stderrScanner} {
			go func(scanner *bufio.Scanner) {
				for scanner.Scan() {
					bufch <- scanner.Text()
				}
			}(s)
		}
	}()

	donech, errch := func() (chan struct{}, chan error) {
		donech := make(chan struct{})
		errch := make(chan error)
		go func() {
			defer func() {
				close(errch)
				close(donech)
				close(bufch)
			}()

			if err := cmd.Start(); err != nil {
				errch <- err
				return
			}
			if err := cmd.Wait(); err != nil {
				errch <- err
				return
			}
		}()
		return donech, errch
	}()

	buf := []string{}
	for {
		select {
		case b := <-bufch:
			buf = append(buf, b)
		case err := <-errch:
			return strings.Join(buf, "\n"), err
		case <-donech:
			goto END
		}
	}
END:

	return strings.Join(buf, "\n"), nil
}

func selectStates(proj string) ([]string, error) {
	out, err := terraform(proj, "state", "list")
	if err != nil {
		return []string{}, fmt.Errorf("%s\n%v", out, err)
	}
	states := strings.Split(out, "\n")

	idx, err := fuzzyfinder.FindMulti(
		states,
		func(i int) string {
			return states[i]
		},
	)
	if err != nil {
		return []string{}, err
	}

	ret := make([]string, 0, len(idx))
	for _, i := range idx {
		ret = append(ret, states[i])
	}
	return ret, nil
}
