import * as bash     from '#infrastructure/bash.js'
import * as logging  from '#infrastructure/logging.js'
import { variables } from '#infrastructure/envVariables.js'
import { secrets }   from '#infrastructure/envSecrets.js'
import axios         from 'axios'

variables.requireEnvVariables([
    'repositoryPath'
])

secrets.require([
    'ghToken'
])

async function sendRepositoryDispatches(githubOrg, eventType, payload) { 
    const repos = await bash.json(`
        gh repo list ${githubOrg} --json name
    `)

    for (const repo of repos) {
        const response = await axios.post(`https://api.github.com/repos/${githubOrg}/${repo.name}/dispatches`, {
            event_type:     eventType,
            client_payload: payload
        }, {
            headers: {
                Accept: 'application/vnd.github+json',
                Authorization: `Bearer ${secrets.ghToken}`,
                "X-GitHub-Api-Version": '2022-11-28'
            }
        })

        console.log(`Sent [${eventType}] repository dispatch to [${repo.name}] with data [${JSON.stringify(payload)}]`)
    }
}

async function main() {
    try {
        logging.printEnvironmentVariables(variables)

        logging.header("Building miruken")

        await sendRepositoryDispatches('Miruken-Go', 'miruken-version-created', {
            version: '0.0.0'
        })

        console.log("Script completed successfully")
    } catch (error) {
        process.exitCode = 1
        console.log(error)
        console.log("Script Failed")
    }
}

main()
