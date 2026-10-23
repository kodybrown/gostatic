// dotask: 1
// description: Build and verify gostatic, then create a self-extracting Windows x64 installer.
// options:
//   - {name: output, type: path, description: Alternate installer output parent directory.}
//   - {name: builder, description: Alternate prebuilt doinstall-builder command or executable path.}
//   - {name: installer, type: path, description: Actual matching doinstall runtime executable.}
//   - {name: launcher-template, type: path, description: Actual matching doinstall launcher executable.}
// requires: [{kind: task, value: stage-installer}, {kind: task, value: _/doinstall/create-installer}, {kind: task, value: _/doinstall/assemble-self-extracting}]
// end-dotask
use dotask_sdk::{BuildContext, Context, Result, json, serde_json};
use std::fs;

fn main() {
    dotask_sdk::run(task);
}

fn task(project: &BuildContext) -> Result<()> {
    if !cfg!(windows) || project.architecture() != "x64" {
        dotask_sdk::bail!("This installer recipe currently targets Windows x64.");
    }

    // The existing generated Go metadata owns the version. Packaging never
    // increments it or rewrites version.go, .version, or version/Version.go.
    let source = fs::read_to_string(project.path("version.go"))?;
    let literals: Vec<_> = source
        .lines()
        .filter_map(|line| line.split_once('='))
        .filter(|(name, _)| name.trim() == "buildVersionStr")
        .map(|(_, literal)| literal.trim())
        .collect();
    if literals.len() != 1 {
        dotask_sdk::bail!("version.go must define exactly one quoted buildVersionStr.");
    }
    let version: String = serde_json::from_str(literals[0])
        .context("buildVersionStr must be a quoted version string")?;
    let version = version.strip_prefix('v').unwrap_or(&version);
    if version.is_empty() {
        dotask_sdk::bail!("buildVersionStr cannot be empty.");
    }

    let mut parameters = json!({"app-version": version});
    if let Ok(builder) = project.string("builder") {
        parameters["builder"] = json!(builder);
    }
    for option in ["output", "installer", "launcher-template"] {
        if let Ok(value) = project.string(option) {
            parameters[option] = json!(project.path(value));
        }
    }
    // The shared adapter queries the installed builder for its real companion
    // tools, stages outside the checkout, and consumes the builder's result JSON.
    let artifact = project.create_installer("_/doinstall/create-installer", parameters)?;
    println!(
        "Installer created: {}",
        artifact["FilePath"]
            .as_str()
            .context("Missing installer FilePath")?
    );
    project.set_installer_result(&artifact)
}
