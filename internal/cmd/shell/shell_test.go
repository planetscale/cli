package shell

import (
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestIsLinked(t *testing.T) {
	c := qt.New(t)

	var tests = []struct {
		name   string
		input  string
		linked bool
	}{
		{
			name:   "darwin linked against libedit",
			linked: true,
			input: `/usr/local/bin/mysql:
        /usr/lib/libedit.3.dylib (compatibility version 2.0.0, current version 3.0.0)
        /usr/local/opt/openssl@1.1/lib/libssl.1.1.dylib (compatibility version 1.1.0, current version 1.1.0)
        /usr/local/opt/openssl@1.1/lib/libcrypto.1.1.dylib (compatibility version 1.1.0, current version 1.1.0)
        /usr/lib/libresolv.9.dylib (compatibility version 1.0.0, current version 1.0.0)
        /usr/lib/libc++.1.dylib (compatibility version 1.0.0, current version 904.4.0)
        /usr/lib/libSystem.B.dylib (compatibility version 1.0.0, current version 1292.60.1)`,
		},
		{
			name:   "linux not linked",
			linked: false,
			input: `linux-vdso.so.1 (0x00007ffc771d7000)
        libpthread.so.0 => /lib64/libpthread.so.0 (0x00007fd23b1ed000)
        librt.so.1 => /lib64/librt.so.1 (0x00007fd23b1e2000)
        libcrypto.so.1.1 => /home/foo/vt/mysql/bin/../lib/private/libcrypto.so.1.1 (0x00007fd23ad1f000)
        libssl.so.1.1 => /home/foo/vt/mysql/bin/../lib/private/libssl.so.1.1 (0x00007fd23aa8e000)
        libdl.so.2 => /lib64/libdl.so.2 (0x00007fd23aa87000)
        libncurses.so.5 => /lib64/libncurses.so.5 (0x00007fd23aa5e000)
        libtinfo.so.5 => /lib64/libtinfo.so.5 (0x00007fd23aa2f000)
        libstdc++.so.6 => /lib64/libstdc++.so.6 (0x00007fd23a810000)
        libm.so.6 => /lib64/libm.so.6 (0x00007fd23a6cc000)
        libgcc_s.so.1 => /lib64/libgcc_s.so.1 (0x00007fd23a6b1000)
        libc.so.6 => /lib64/libc.so.6 (0x00007fd23a4e2000)
        /lib64/ld-linux-x86-64.so.2 (0x00007fd23b226000) `,
		},
		{
			name:   "linux linked against readline",
			linked: true,
			input: ` linux-vdso.so.1 (0x00007ffde1ed8000)
        libpthread.so.0 => /lib64/libpthread.so.0 (0x00007f6cd29ee000)
        libreadline.so.6.0 => /home/foo/../lib/private/libreadline.so.6.0 (0x00007f6cd27a3000)
        libtinfo.so.5.7 => /home/foo/../lib/private/libtinfo.so.5.7 (0x00007f6cd2580000)
        libz.so.1 => /lib64/libz.so.1 (0x00007f6cd2566000)
        librt.so.1 => /lib64/librt.so.1 (0x00007f6cd255b000)
        libssl.so.1.0.1e => /home/foo/../lib/private/libssl.so.1.0.1e (0x00007f6cd22d5000)
        libcrypto.so.1.0.1e => /home/foo/../lib/private/libcrypto.so.1.0.1e (0x00007f6cd1ed1000)
        libdl.so.2 => /lib64/libdl.so.2 (0x00007f6cd1eca000)
        libm.so.6 => /lib64/libm.so.6 (0x00007f6cd1d86000)
        libstdc++.so.6 => /lib64/libstdc++.so.6 (0x00007f6cd1b67000)
        libgcc_s.so.1 => /lib64/libgcc_s.so.1 (0x00007f6cd1b4c000)
        libc.so.6 => /lib64/libc.so.6 (0x00007f6cd197d000)
        /lib64/ld-linux-x86-64.so.2 (0x00007f6cd2a27000)
        libgssapi_krb5.so.2.2 => /home/foo/../lib/private/libgssapi_krb5.so.2.2 (0x00007f6cd172b000)
        libkrb5.so.3.3 => /home/foo/../lib/private/libkrb5.so.3.3 (0x00007f6cd1434000)
        libcom_err.so.2 => /lib64/libcom_err.so.2 (0x00007f6cd142d000)
        libk5crypto.so.3.1 => /home/foo/../lib/private/libk5crypto.so.3.1 (0x00007f6cd11fe000)
        libkrb5support.so.0.1 => /home/foo/../lib/private/libkrb5support.so.0.1 (0x00007f6cd0ff2000)
        libkeyutils.so.1 => /lib64/libkeyutils.so.1 (0x00007f6cd0fe9000)
        libresolv.so.2 => /lib64/libresolv.so.2 (0x00007f6cd0fcf000)
        libselinux.so.1 => /lib64/libselinux.so.1 (0x00007f6cd0fa3000)
        libpcre2-8.so.0 => /lib64/libpcre2-8.so.0 (0x00007f6cd0f0c000)`,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			linked := isLinked(tt.input)
			c.Assert(linked, qt.Equals, tt.linked)
		})
	}

}
