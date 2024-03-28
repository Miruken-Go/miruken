"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const ci_cd_1 = require("ci.cd");
(0, ci_cd_1.handle)(async () => {
    const variables = new ci_cd_1.EnvVariables()
        .required([
        'repositoryPath',
        'repository',
        'repositoryOwner',
        'ref',
    ])
        .optional(['skipRepositoryDispatches'])
        .variables;
    ci_cd_1.logging.printVariables(variables);
    const secrets = new ci_cd_1.EnvSecrets()
        .require(['GH_TOKEN'])
        .secrets;
    ci_cd_1.logging.printSecrets(secrets);
    ci_cd_1.logging.header("Building miruken");
    await ci_cd_1.bash.execute(`
        go test ./...
    `);
    //This docker container is running docker in docker from github actions
    //Therefore using $(pwd) to get the working directory would be the working directory of the running container 
    //Not the working directory from the host system. So we need to pass in the repository path.
    const rawVersion = await ci_cd_1.bash.execute(`
        docker run --rm -v '${variables.repositoryPath}:/repo' \
        gittools/gitversion:5.12.0-alpine.3.14-6.0 /repo /showvariable SemVer
    `);
    const gitTag = `v${rawVersion}`;
    await new ci_cd_1.Git(secrets.GH_TOKEN)
        .tagAndPush(gitTag);
    await new ci_cd_1.GH({
        ghToken: secrets.GH_TOKEN,
        ref: variables.ref,
        repository: variables.repository,
        repositoryOwner: variables.repositoryOwner,
        skipRepositoryDispatches: Boolean(variables.skipRepositoryDispatches)
    }).sendRepositoryDispatches('built-miruken', {
        mirukenVersion: gitTag
    });
});
