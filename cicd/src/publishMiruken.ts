import {
    bash,
    logging,
    Git,
    handle,
    EnvVariables,
    EnvSecrets,
    GH
} from 'ci.cd'

handle(async () => {

    const variables = new EnvVariables()
        .required([
            'repositoryPath',
            'repository',
            'repositoryOwner',
            'ref',
        ])
        .optional(['skipRepositoryDispatches'])
        .variables
    logging.printVariables(variables)

    const secrets = new EnvSecrets()
        .require(['GH_TOKEN'])
        .secrets
    logging.printSecrets(secrets)

    logging.header("Running tests")

    await bash.execute(`
        cd ../
        go test ./...
    `)

    logging.header("Publishing new miruken version ")

    //This docker container is running docker in docker from github actions
    //Therefore using $(pwd) to get the working directory would be the working directory of the running container 
    //Not the working directory from the host system. So we need to pass in the repository path.
    const rawVersion = await bash.execute(`
        docker run --rm -v '${variables.repositoryPath}:/repo' \
        gittools/gitversion:6.0.0-alpine.3.18-7.0 /repo /showvariable SemVer
    `)

    const gitTag = `v${rawVersion}`

    await new Git(secrets.GH_TOKEN)
        .tagAndPush(gitTag)

    await new GH({
        ghToken:                  secrets.GH_TOKEN,
        ref:                      variables.ref,
        repository:               variables.repository,
        repositoryOwner:          variables.repositoryOwner,
        skipRepositoryDispatches: Boolean(variables.skipRepositoryDispatches)
    }).sendRepositoryDispatches('built-miruken', {
        mirukenVersion: gitTag
    })
})
