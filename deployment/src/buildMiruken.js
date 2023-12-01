import * as bash     from '#infrastructure/bash.js'
import * as logging  from '#infrastructure/logging.js'
import * as git      from '#infrastructure/git.js'
import * as gh       from '#infrastructure/gh.js'
import { variables } from '#infrastructure/envVariables.js'

variables.requireEnvVariables([
    'repositoryPath'
])

variables.optionalEnvVariables([
    'skipRepositoryDispatches'
])

async function main() {
    try {
        logging.printEnvironmentVariables(variables)

        logging.header("Building miruken")

        await bash.execute(`
            go test ./...
        `)

        //This docker container is running docker in docker from github actions
        //Therefore using $(pwd) to get the working directory would be the working directory of the running container 
        //Not the working directory from the host system. So we need to pass in the repository path.
        const rawVersion = await bash.execute(`
            docker run --rm -v '${variables.repositoryPath}:/repo' \
            gittools/gitversion:5.12.0-alpine.3.14-6.0 /repo /showvariable SemVer
        `)

        const gitTag = `v${rawVersion}`

        console.log(`gitTag: [${gitTag}]`)

        await git.tagAndPush(gitTag)

        if (!variables.skipRepositoryDispatches) {
            await gh.sendRepositoryDispatches('Miruken-Go', 'miruken-version-created', {
                version: gitTag
            })
        }

        console.log("Script completed successfully")
    } catch (error) {
        process.exitCode = 1
        console.log(error)
        console.log("Script Failed")
    }
}

main()
