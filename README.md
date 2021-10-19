# Snapr

[![Build](https://github.com/ckornie/snapr/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/ckornie/snapr/actions/workflows/build.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/ckornie/snapr)](https://goreportcard.com/report/github.com/ckornie/snapr)

## Introduction
Snapr creates ZFS snapshots on a schedule and replicates them to S3 compatible storage. It aims to be simple and sturdy.  Use it if you want to create snapshots on a schedule, replicate them, and restore at a later date.

## Background
I wanted to forward ZFS streams generated from [zfs-send](https://openzfs.github.io/openzfs-docs/man/8/zfs-send.8.html) to S3 compatible storage. My requirements:

- Use the raw mode of `zfs send` (the file systems I replicate use native encryption and compression).
- Replicate and restore without creating interim files.
- Straight-forward configuration.
- A high level of reliability.

I could not find a foolproof way to do this. I initially looked at writing a short shell script to utilize existing utilities. I hit a few road blocks. The official Backblaze client doesn't support sending from stdin (see: [#152](https://github.com/Backblaze/B2_Command_Line_Tool/issues/152)). Using [s3cmd](https://s3tools.org/s3cmd) does work:

```console
root@example ~ # zfs send -w pool-0/example@daily-00010 | s3cmd put - s3://big-bucket/pool-0/example/daily-00010
```

However in my trials it was slow (doesn't upload parts concurrently) and error prone.

As I was looking to learn Go I decided to write a simple tool to meet my requirements.

## Usage
Snapr has three primary functions. It snaps (creates snapshots), sends (forwards a ZFS stream to S3 compatible storage), and restores. Scheduling works by combining your job scheduler (i.e. cron, systemd timers) with a per file system interval.

Let's take a look at a simple configuration for the file system 'pool-0/example' and then look at how it pertains to each of the functions.

```json
{
  "fileSystems": {
    "pool-0/example": {
      "snap":[
        {
          "interval": "23h30m",
          "prefix": "daily",
          "hold": [
            "aws"
          ]
        },
        {
          "interval": "30m",
          "prefix": "hourly"
        }
      ],
      "send": [
        {
          "endpoint": "s3.us‑east‑1.amazonaws.com",
          "region": "us‑east‑1",
          "account": "0000000000001",
          "secret": "passw0rd",
          "bucket": "big-bucket",
          "release": [
            "aws"
          ]
        }
      ]
    }
  },
  "threads": 20,
  "volumeSize": 200000,
  "partSize": 200
}
```

### Snap
Snapr will take snapshots when run with the `--snap` argument. You can see the above configuration lists an interval, prefix, and optional hold for each entry in the 'snap' list. The prefix is used for snapshot naming (e.g. 'daily-00010'). Interval is used as a minimum time between snapshots. Hold is used to specify holds which are to be applied to new snapshots (see [zfs-hold](https://openzfs.github.io/openzfs-docs/man/8/zfs-hold.8.html)).

Let's take a look at the 'daily' snap. In this case a snapshot will be taken if there's no snapshot using the 'daily' prefix within the last 23 hours and 30 minutes. The next snapshot generated will be 'daily-n' where n is a sequential number starting from 0. The interval is specified in Go's [duration string format](https://pkg.go.dev/time#ParseDuration).

If you schedule `snapr --snap` to run every hour then a snapshot will be taken once each run for the 'hourly' set and once daily for the 'daily' set.

### Send
Snapr will send streams when run with the `--send` argument. A full stream will be sent if there are no prior archives at the send destination and an incremental will be sent otherwise. The incremental streams snapr generates are equivalent to:

```console
root@example ~ # zfs send --raw --replicate --holds -I <source> <target>
```

Let's assume snapr generated an incremental stream using the configuration above from `pool-0/example@hourly-00005` (source) to `pool-0/example@hourly-00010` (target). If the destination has one prior archive (00000) consisting of a full stream then this command would:

- Break the incremental stream into volumes of 200 GB and send them to to 'big-bucket'. The first volume would be 'pool-0/example/00001/00000'.
- Remove all holds of 'aws' from the source snapshot (hourly-00005) and following intermediary snapshots.
- Maintain the existing hold on the target snapshot (hourly-00010). This is a safeguard to prevent it from being deleted such that it can be used as a source for the next incremental send.
- Write the archive contents to 'pool-0/example/00001/contents'.

Once successfully sent only the target snapshot must be maintained. All other snapshots can be destroyed.

You can define multiple send entries if you require region or provider redundancy. The final entry will be used to restore.

Snapr utilizes [multi-part uploads](https://docs.aws.amazon.com/AmazonS3/latest/userguide/mpuoverview.html) to improve performance. There are two settings exposed for tuning. The `threads` setting indicates how many parts will be sent in parallel. The `partSize` (megabytes) is the size of each part.

The `volumeSize` (megabytes) specifies the maximum size for a single file. Cloud providers usually have a maximum (e.g. Amazon's is [5 terabytes](https://aws.amazon.com/s3/faqs/)). These settings can be set globally and overridden per send entry.

### Restore
To restore a file system it must be configured. The following command will perform a full restore of the file system `pool-0/example`:

```console
root@example ~ # snapr --restore --filesystem "pool-0/example"
```

This will look at the configured 'send' entries for `pool-0/example` and restore from the **last** entry. It will fail if the file system is already present locally. You can perform a rename if you want to keep an existing file system:

```console
root@example ~ # zfs rename pool-0/example pool-0/example-defunct
```

The restore will download and receive all available archives incrementally. Volumes will be downloaded in parts according to the specified `partSize`.

Once restored, encrypted file systems will default to prompting for their keys. To set the keys to be inherited from the parent you can do the following:

```console
root@example ~ # zfs load-key pool-0/example
Enter passphrase for 'pool-0/example':
root@example ~ # zfs change-key -i -l pool-0/example
root@example ~ # zfs mount pool-0/example
```

### Policy Based Snapshots
As my requirements are very simple I haven't implemented policy based snapshots. If you need more complex snapshot scheduling you can look towards:

- [sanoid](https://github.com/jimsalterjrs/sanoid)
- [zfs-auto-snapshot](https://github.com/zfsonlinux/zfs-auto-snapshot)

Both of these tools - in addition to creating snapshots - can destroy ZFS snapshots according to a policy. The send functionality of Snapr should work smoothly with these tools. Simply leave your 'snap' list empty.
