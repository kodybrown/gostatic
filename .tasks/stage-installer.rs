// dotask: 1
// description: Verify and stage the exact native gostatic payload for installer creation.
// options:
//   - {name: stage-dir, type: path, required: true, description: Private external installer staging directory.}
//   - {name: app-version, required: true, description: Expected version from cmd/gostatic/version.go.}
// requires: [{kind: tool, value: go}]
// end-dotask
use dotask_sdk::{BuildContext, Result};
use std::{env, fs};

fn main() {
    dotask_sdk::run(task);
}

fn task(project: &BuildContext) -> Result<()> {
    if !cfg!(windows) || project.architecture() != "x64" {
        dotask_sdk::bail!("This installer recipe currently targets Windows x64.");
    }
    let stage = project.path(project.string("stage-dir")?);
    if stage.exists() {
        dotask_sdk::bail!(
            "Installer staging directory must be fresh: {}",
            stage.display()
        );
    }
    fs::create_dir_all(&stage)?;

    // Explicit native targets prevent an inherited cross-compilation setting
    // from producing a payload that cannot run with the selected installer.
    for operation in ["test", "vet"] {
        let mut command = project.command("go");
        command.args([operation, "./..."]);
        command.env("GOOS", "windows").env("GOARCH", "amd64");
        project.execute(&mut command)?;
    }
    let executable = stage.join("gostatic.exe");
    project.execute(
        project
            .command("go")
            .args(["build", "-trimpath", "-o"])
            .arg(&executable)
            .arg("./cmd/gostatic")
            .env("GOOS", "windows")
            .env("GOARCH", "amd64"),
    )?;

    // Refuse a distributable containing local checkout/profile paths. -trimpath
    // removes compiler paths; this catches accidental path embedding as well.
    let bytes = fs::read(&executable)?;
    let mut private_paths = vec![project.root().to_string_lossy().into_owned()];
    if let Some(home) = env::var_os("USERPROFILE") {
        private_paths.push(home.to_string_lossy().into_owned());
    }
    for path in private_paths {
        for spelling in [path.replace('\\', "/"), path.replace('/', "\\")] {
            if !spelling.is_empty()
                && bytes
                    .windows(spelling.len())
                    .any(|part| part == spelling.as_bytes())
            {
                dotask_sdk::bail!(
                    "The compiled executable contains a local checkout/profile path."
                );
            }
        }
    }

    let mut version_command = project.command(&executable);
    version_command.arg("--version").current_dir(&stage);
    // A packaging smoke check must not inherit site-specific settings.
    for (name, _) in env::vars_os() {
        if name
            .to_string_lossy()
            .to_ascii_uppercase()
            .starts_with("GOSTATIC_")
        {
            version_command.env_remove(name);
        }
    }
    let actual = project.capture(&mut version_command)?;
    let expected = format!("gostatic v{}", project.string("app-version")?);
    if actual.trim() != expected {
        dotask_sdk::bail!("Built executable version does not match installer metadata.");
    }
    for filename in ["README.md", "LICENSE.md"] {
        fs::copy(project.path(filename), stage.join(filename))?;
    }
    println!("Verified installer payload: {expected}");
    Ok(())
}
