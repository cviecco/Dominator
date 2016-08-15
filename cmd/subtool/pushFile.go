package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	objclient "github.com/Symantec/Dominator/lib/objectserver/client"
	"github.com/Symantec/Dominator/lib/triggers"
	"github.com/Symantec/Dominator/proto/sub"
	"github.com/Symantec/Dominator/sub/client"
	"os"
	"syscall"
)

func pushFileSubcommand(getSubClient getSubClientFunc, args []string) {
	if err := pushFile(getSubClient, args[0], args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "Error pushing file: %s\n", err)
		os.Exit(2)
	}
	os.Exit(0)
}

func pushFile(getSubClient getSubClientFunc, source, dest string) error {
	var sourceStat syscall.Stat_t
	if err := syscall.Stat(source, &sourceStat); err != nil {
		return err
	}
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	srpcClient := getSubClient()
	objClient := objclient.AttachObjectClient(srpcClient)
	defer objClient.Close()
	if err != nil {
		return err
	}
	hashVal, _, err := objClient.AddObject(sourceFile, uint64(sourceStat.Size),
		nil)
	if err != nil {
		return err
	}
	newRegularInode := &filesystem.RegularInode{
		Mode:             filesystem.FileMode(sourceStat.Mode),
		Uid:              sourceStat.Uid,
		Gid:              sourceStat.Gid,
		MtimeNanoSeconds: int32(sourceStat.Mtim.Nsec),
		MtimeSeconds:     sourceStat.Mtim.Sec,
		Size:             uint64(sourceStat.Size),
		Hash:             hashVal}
	newInode := sub.Inode{Name: dest, GenericInode: newRegularInode}
	var updateRequest sub.UpdateRequest
	var updateReply sub.UpdateResponse
	updateRequest.Wait = true
	updateRequest.InodesToMake = append(updateRequest.InodesToMake, newInode)
	if *triggersFile != "" {
		updateRequest.Triggers, err = triggers.Load(*triggersFile)
		if err != nil {
			return err
		}
	} else if *triggersString != "" {
		updateRequest.Triggers, err = triggers.Decode([]byte(*triggersString))
		if err != nil {
			return err
		}
	}
	return client.CallUpdate(srpcClient, updateRequest, &updateReply)
}
