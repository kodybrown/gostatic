// dotask: 1
// description: Exercise a Windows x64 gostatic installer in isolated temporary directories.
// options:
//   - {name: installer, type: path, required: true, description: Self-extracting gostatic installer to test.}
// end-dotask
use dotask_sdk::{BuildContext, Context, Result, tempfile};
use std::{
    env,
    ffi::{OsStr, OsString},
    fs,
    path::{Path, PathBuf},
    process::{Command, Stdio},
    thread,
    time::{Duration, Instant},
};

fn main() {
    dotask_sdk::run(task);
}

fn task(project: &BuildContext) -> Result<()> {
    if !cfg!(windows) || project.architecture() != "x64" {
        dotask_sdk::bail!("Installer acceptance currently targets Windows x64.");
    }
    let installer = fs::canonicalize(project.path(project.string("installer")?))
        .context("Installer must name an existing self-extracting executable")?;
    if !installer.is_file() {
        dotask_sdk::bail!("Installer must be a file: {}", installer.display());
    }
    let before = persistent_paths()?;
    // Retain failed fixtures for diagnosis. Only remove a successful fixture
    // after proving that its canonical path stays inside the OS temporary root.
    let root = tempfile::Builder::new()
        .prefix("gostatic-installer-test-")
        .suffix(" space")
        .tempdir()?
        .keep();
    let mut acceptance = Acceptance {
        root,
        invocation: 0,
    };
    let result = acceptance.check(&installer);
    // Inspect the real registry values, including type and unexpanded contents,
    // rather than just the current process's inherited PATH snapshot.
    let path_check = persistent_paths().and_then(|after| {
        if before != after {
            dotask_sdk::bail!("Installer acceptance changed persistent user/machine PATH.");
        }
        Ok(())
    });
    if result.is_err() || path_check.is_err() {
        eprintln!(
            "Failed acceptance fixture retained: {}",
            acceptance.root.display()
        );
        // Report a PATH change even if an earlier acceptance check also failed.
        if let Err(error) = path_check {
            if let Err(original) = result {
                return Err(error.context(format!("Acceptance also failed: {original:#}")));
            }
            return Err(error);
        }
        return result;
    }
    let resolved = acceptance.root.canonicalize()?;
    let temporary = env::temp_dir().canonicalize()?;
    let name = resolved
        .file_name()
        .context("Missing fixture directory name")?
        .to_string_lossy();
    if !resolved.starts_with(&temporary)
        || !name.starts_with("gostatic-installer-test-")
        || !name.ends_with(" space")
    {
        dotask_sdk::bail!("Unexpected cleanup target: {}", resolved.display());
    }
    fs::remove_dir_all(&resolved)?;
    println!("PASS: user/machine PATH unchanged; temporary acceptance files cleaned up.");
    Ok(())
}

struct Acceptance {
    root: PathBuf,
    invocation: usize,
}

impl Acceptance {
    fn run(
        &mut self,
        executable: &Path,
        arguments: impl IntoIterator<Item = impl AsRef<OsStr>>,
    ) -> Result<String> {
        self.invocation += 1;
        let stdout = self
            .root
            .join(format!("process-{}.stdout", self.invocation));
        let stderr = self
            .root
            .join(format!("process-{}.stderr", self.invocation));
        let mut command = Command::new(executable);
        command
            .args(arguments)
            .current_dir(&self.root)
            .stdin(Stdio::null())
            // Files avoid pipe-reader deadlocks if a failed installer emits a
            // large diagnostic. Keep those logs with any failed fixture.
            .stdout(fs::File::create(&stdout)?)
            .stderr(fs::File::create(&stderr)?)
            .env("APPDATA", self.root.join("roaming"))
            .env("LOCALAPPDATA", self.root.join("local"));
        for (name, _) in env::vars_os() {
            if name
                .to_string_lossy()
                .to_ascii_uppercase()
                .starts_with("GOSTATIC_")
            {
                command.env_remove(name);
            }
        }
        #[cfg(windows)]
        {
            use std::os::windows::process::CommandExt;
            command.creation_flags(0x08000000); // CREATE_NO_WINDOW
        }
        let mut child = command
            .spawn()
            .with_context(|| format!("Start {}", executable.display()))?;
        let deadline = Instant::now() + Duration::from_secs(30);
        let status = loop {
            match child.try_wait() {
                Ok(Some(status)) => break status,
                Ok(None) => {}
                Err(error) => {
                    let _ = child.kill();
                    let _ = child.wait();
                    return Err(error.into());
                }
            }
            if Instant::now() >= deadline {
                let _ = child.kill();
                let _ = child.wait();
                dotask_sdk::bail!(
                    "Timed out: {} (logs under {})",
                    executable.display(),
                    self.root.display()
                );
            }
            thread::sleep(Duration::from_millis(25));
        };
        let output = fs::read_to_string(&stdout)?;
        if !status.success() {
            dotask_sdk::bail!(
                "{status} from {}:\n{}{}",
                executable.display(),
                output,
                fs::read_to_string(&stderr)?
            );
        }
        Ok(output)
    }

    fn install(&mut self, installer: &Path, extraction: &str, validate: bool) -> Result<()> {
        let mut args: Vec<OsString> = vec![
            "--extract-to".into(),
            self.root.join(extraction).into_os_string(),
            "--non-interactive".into(),
            "--install-dir".into(),
            self.root.join("application").into_os_string(),
            "--set".into(),
            "add-to-path=false".into(),
            "--set".into(),
            "additional-command=false".into(),
        ];
        if validate {
            args.push("--validate".into());
        }
        self.run(installer, args)?;
        Ok(())
    }

    fn check(&mut self, installer: &Path) -> Result<()> {
        let application = self.root.join("application");
        self.install(installer, "validation extraction", true)?;
        if application.exists() {
            dotask_sdk::bail!("Validation created an installation.");
        }
        self.install(installer, "initial extraction", false)?;
        let executable = application.join("gostatic.exe");
        let version = self.run(&executable, ["--version"])?;
        let version = version.trim();
        if !version
            .strip_prefix("gostatic v")
            .and_then(|v| v.chars().next())
            .is_some_and(|c| c.is_ascii_digit())
        {
            dotask_sdk::bail!("Unexpected installed version: {version}");
        }
        let builds: Vec<_> = fs::read_dir(application.join("app"))?
            .collect::<std::io::Result<Vec<_>>>()?
            .into_iter()
            .filter(|entry| entry.path().is_dir())
            .collect();
        if builds.len() != 1 {
            dotask_sdk::bail!("Expected one installed application build.");
        }
        let mut payload = Vec::new();
        for entry in fs::read_dir(builds[0].path())? {
            let entry = entry?;
            let name = entry.file_name();
            if entry.file_type()?.is_file()
                && name != "installation.yaml"
                && name != "installer.yaml"
            {
                payload.push(name);
            }
        }
        payload.sort();
        let mut expected: Vec<OsString> = ["gostatic.exe", "README.md", "LICENSE.md"]
            .map(OsString::from)
            .into();
        expected.sort();
        if payload != expected {
            dotask_sdk::bail!("Unexpected installed payload: {payload:?}");
        }
        reject_site_config(&application)?;

        self.install(installer, "reinstall extraction", false)?;
        if self.run(&executable, ["--version"])?.trim() != version {
            dotask_sdk::bail!("Reinstall changed the application version.");
        }
        self.run(
            &application.join("installer/installer.exe"),
            ["uninstall", "--non-interactive", "--leave-settings"],
        )?;
        if executable.exists() {
            dotask_sdk::bail!("Uninstall left the application command behind.");
        }
        println!(
            "PASS: validation, install, exact payload, installed version ({version}), reinstall, and uninstall."
        );
        Ok(())
    }
}

fn reject_site_config(directory: &Path) -> Result<()> {
    for entry in fs::read_dir(directory)? {
        let entry = entry?;
        let kind = entry.file_type()?;
        if kind.is_symlink() {
            dotask_sdk::bail!(
                "Unexpected link in acceptance installation: {}",
                entry.path().display()
            );
        }
        if kind.is_file()
            && entry
                .file_name()
                .to_string_lossy()
                .eq_ignore_ascii_case(".gostatic")
        {
            dotask_sdk::bail!("Installer copied a site-specific .gostatic file.");
        }
        if kind.is_dir() {
            reject_site_config(&entry.path())?;
        }
    }
    Ok(())
}

type RegistryValue = Option<(u32, Vec<u8>)>;

#[cfg(windows)]
fn persistent_paths() -> Result<[RegistryValue; 2]> {
    use std::{ffi::c_void, ptr};
    #[link(name = "advapi32")]
    unsafe extern "system" {
        fn RegGetValueW(
            key: *mut c_void,
            subkey: *const u16,
            value: *const u16,
            flags: u32,
            kind: *mut u32,
            data: *mut c_void,
            size: *mut u32,
        ) -> i32;
    }
    fn read(key: u32, subkey: &str) -> Result<RegistryValue> {
        let subkey: Vec<u16> = subkey.encode_utf16().chain(Some(0)).collect();
        let value: Vec<u16> = "Path".encode_utf16().chain(Some(0)).collect();
        let key = key as i32 as isize as *mut c_void;
        let mut kind = 0;
        let mut size = 0;
        let flags = 0x10000006; // RRF_NOEXPAND | RRF_RT_REG_SZ | RRF_RT_REG_EXPAND_SZ
        // Safety: fixed predefined registry roots and NUL-terminated UTF-16
        // names; Windows writes only to the supplied sized buffers.
        let status = unsafe {
            RegGetValueW(
                key,
                subkey.as_ptr(),
                value.as_ptr(),
                flags,
                &mut kind,
                ptr::null_mut(),
                &mut size,
            )
        };
        if status == 2 || status == 3 {
            return Ok(None);
        }
        if status != 0 {
            return Err(std::io::Error::from_raw_os_error(status).into());
        }
        let mut data = vec![0u8; size as usize];
        let status = unsafe {
            RegGetValueW(
                key,
                subkey.as_ptr(),
                value.as_ptr(),
                flags,
                &mut kind,
                data.as_mut_ptr().cast(),
                &mut size,
            )
        };
        if status != 0 {
            return Err(std::io::Error::from_raw_os_error(status).into());
        }
        data.truncate(size as usize);
        Ok(Some((kind, data)))
    }
    Ok([
        read(0x80000001, "Environment")?,
        read(
            0x80000002,
            r"SYSTEM\CurrentControlSet\Control\Session Manager\Environment",
        )?,
    ])
}

#[cfg(not(windows))]
fn persistent_paths() -> Result<[RegistryValue; 2]> {
    dotask_sdk::bail!("Persistent PATH verification requires Windows.")
}
