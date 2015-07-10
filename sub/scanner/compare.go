package scanner

import (
	"bytes"
	"fmt"
	"io"
	"syscall"
)

func compare(left, right *FileSystem, logWriter io.Writer) bool {
	if len(left.RegularInodeTable) != len(right.RegularInodeTable) {
		if logWriter != nil {
			fmt.Fprintf(logWriter,
				"left vs. right: %d vs. %d regular file inodes\n",
				len(left.RegularInodeTable), len(right.RegularInodeTable))
		}
		return false
	}
	if len(left.InodeTable) != len(right.InodeTable) {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "left vs. right: %d vs. %d inodes\n",
				len(left.InodeTable), len(right.InodeTable))
		}
		return false
	}
	if len(left.DirectoryInodeList) != len(right.DirectoryInodeList) {
		if logWriter != nil {
			fmt.Fprintf(logWriter,
				"left vs. right: %d vs. %d directory inodes\n",
				len(left.DirectoryInodeList), len(right.DirectoryInodeList))
		}
		return false
	}
	if !compareDirectories(&left.Directory, &right.Directory, logWriter) {
		return false
	}
	if len(left.ObjectCache) != len(right.ObjectCache) {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "left vs. right: %d vs. %d objects\n",
				len(left.ObjectCache), len(right.ObjectCache))
		}
		return false
	}
	return compareObjects(left.ObjectCache, right.ObjectCache, logWriter)
}

func compareDirectories(left, right *Directory, logWriter io.Writer) bool {
	if left.Name != right.Name {
		fmt.Fprintf(logWriter, "dirname: left vs. right: %s vs. %s\n",
			left.Name, right.Name)
		return false
	}
	if left.Mode != right.Mode {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Mode: left vs. right: %x vs. %x\n",
				left.Mode, right.Mode)
		}
		return false
	}
	if left.Uid != right.Uid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Uid: left vs. right: %d vs. %d\n",
				left.Uid, right.Uid)
		}
		return false
	}
	if left.Gid != right.Gid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Gid: left vs. right: %d vs. %d\n",
				left.Gid, right.Gid)
		}
		return false
	}
	if len(left.RegularFileList) != len(right.RegularFileList) {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "left vs. right: %d vs. %d regular files\n",
				len(left.RegularFileList), len(right.RegularFileList))
		}
		return false
	}
	if len(left.FileList) != len(right.FileList) {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "left vs. right: %d vs. %d files\n",
				len(left.FileList), len(right.FileList))
		}
		return false
	}
	if len(left.DirectoryList) != len(right.DirectoryList) {
		fmt.Fprintf(logWriter, "left vs. right: %d vs. %d subdirs\n",
			len(left.DirectoryList), len(right.DirectoryList))
		return false
	}
	for index, leftEntry := range left.RegularFileList {
		if !compareRegularFiles(leftEntry, right.RegularFileList[index],
			logWriter) {
			return false
		}
	}
	for index, leftEntry := range left.FileList {
		if !compareFiles(leftEntry, right.FileList[index], logWriter) {
			return false
		}
	}
	for index, leftEntry := range left.DirectoryList {
		if !compareDirectories(leftEntry, right.DirectoryList[index],
			logWriter) {
			return false
		}
	}
	return true
}

func compareRegularFiles(left, right *RegularFile, logWriter io.Writer) bool {
	if left.Name != right.Name {
		fmt.Fprintf(logWriter, "filename: left vs. right: %s vs. %s\n",
			left.Name, right.Name)
		return false
	}
	if !compareRegularInodes(left.inode, right.inode, logWriter) {
		return false
	}
	return true
}

func compareRegularInodes(left, right *RegularInode, logWriter io.Writer) bool {
	if left.Mode != right.Mode {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Mode: left vs. right: %x vs. %x\n",
				left.Mode, right.Mode)
		}
		return false
	}
	if left.Uid != right.Uid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Uid: left vs. right: %d vs. %d\n",
				left.Uid, right.Uid)
		}
		return false
	}
	if left.Gid != right.Gid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Gid: left vs. right: %d vs. %d\n",
				left.Gid, right.Gid)
		}
		return false
	}
	if left.Size != right.Size {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Size: left vs. right: %d vs. %d\n",
				left.Size, right.Size)
		}
		return false
	}
	if left.Mtime != right.Mtime {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Mtime: left vs. right: %d vs. %d\n",
				left.Mtime, right.Mtime)
		}
		return false
	}
	if bytes.Compare(left.Hash[:], right.Hash[:]) != 0 {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "hash: left vs. right: %x vs. %x\n",
				left.Hash, right.Hash)
		}
		return false
	}
	return true
}

func compareFiles(left, right *File, logWriter io.Writer) bool {
	if left.Name != right.Name {
		fmt.Fprintf(logWriter, "filename: left vs. right: %s vs. %s\n",
			left.Name, right.Name)
		return false
	}
	if !compareInodes(left.inode, right.inode, logWriter) {
		return false
	}
	return true
}

func compareInodes(left, right *Inode, logWriter io.Writer) bool {
	if left.Mode != right.Mode {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Mode: left vs. right: %x vs. %x\n",
				left.Mode, right.Mode)
		}
		return false
	}
	if left.Uid != right.Uid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Uid: left vs. right: %d vs. %d\n",
				left.Uid, right.Uid)
		}
		return false
	}
	if left.Gid != right.Gid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Gid: left vs. right: %d vs. %d\n",
				left.Gid, right.Gid)
		}
		return false
	}
	if left.Size != right.Size {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Size: left vs. right: %d vs. %d\n",
				left.Size, right.Size)
		}
		return false
	}
	if left.Mode&syscall.S_IFMT != syscall.S_IFLNK {
		if left.Mtime != right.Mtime {
			if logWriter != nil {
				fmt.Fprintf(logWriter, "Mtime: left vs. right: %d vs. %d\n",
					left.Mtime, right.Mtime)
			}
			return false
		}
	}
	if left.Mode&syscall.S_IFMT == syscall.S_IFBLK ||
		left.Mode&syscall.S_IFMT == syscall.S_IFCHR {
		if left.Rdev != right.Rdev {
			if logWriter != nil {
				fmt.Fprintf(logWriter, "Rdev: left vs. right: %x vs. %x\n",
					left.Rdev, right.Rdev)
			}
			return false
		}
	}
	if left.Mode&syscall.S_IFMT == syscall.S_IFLNK {
		if left.Symlink != right.Symlink {
			if logWriter != nil {
				fmt.Fprintf(logWriter, "symlink: left vs. right: %s vs. %s\n",
					left.Symlink, right.Symlink)
			}
			return false
		}
	}
	return true
}

func compareObjects(left [][]byte, right [][]byte, logWriter io.Writer) bool {
	for index, leftHash := range left {
		if bytes.Compare(leftHash, right[index]) != 0 {
			if logWriter != nil {
				fmt.Fprintf(logWriter, "hash: left vs. right: %x vs. %x\n",
					leftHash, right[index])
			}
			return false
		}
	}
	return true
}