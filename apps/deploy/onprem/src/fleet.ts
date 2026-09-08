import * as path from 'node:path';
import * as pulumi from '@pulumi/pulumi';
import { local } from '@pulumi/command';
import type { Host } from './hosts';

export function deployFleet(opts: { hosts: Host[]; release: string; repoRoot: string; sshKeyPath: string }): { hostReleases: pulumi.Output<Record<string, string>> } {
  const script = path.join(opts.repoRoot, 'apps/deploy/onprem/scripts/release-host.sh');
  const releases: Record<string, string> = {};

  for (const host of opts.hosts) {
    new local.Command(
      `onprem-${host.id}`,
      {
        dir: opts.repoRoot,
        create: `bash ${script}`,
        update: `bash ${script}`,
        delete: 'true',
        environment: {
          ONPREM_RELEASE: opts.release,
          ONPREM_SSH_KEY_PATH: opts.sshKeyPath,
          ONPREM_HOST_USER: host.user,
          ONPREM_HOST_ADDRESS: host.address,
          ONPREM_HOST_PORT: String(host.port),
          ONPREM_HOST_REMOTE_DIR: host.remoteDir,
        },
        triggers: [opts.release, host.address, host.user, String(host.port), host.remoteDir],
      },
      { retainOnDelete: true },
    );
    releases[host.id] = opts.release;
  }

  return { hostReleases: pulumi.output(releases) };
}
