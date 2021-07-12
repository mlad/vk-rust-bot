import subprocess
import os

def go_build(go_os, go_arch, filename):
    env = dict(os.environ)
    env['GOOS'] = go_os
    env['GOARCH'] = go_arch

    print('Build:', go_os, go_arch, filename)
    try:
        subprocess.run(('go', 'build', '-trimpath', '-ldflags=-s -w', '-o=' + filename), env=env, check=True)
    except subprocess.CalledProcessError:
        print('Compile error')
        return

def main():
    go_build('linux', 'amd64', 'linux_x64')
    go_build('linux', '386', 'linux_x32')
    go_build('windows', 'amd64', 'windows_x64.exe')
    go_build('windows', '386', 'windows_x32.exe')
    print('Complete')


if __name__ == '__main__':
    main()
