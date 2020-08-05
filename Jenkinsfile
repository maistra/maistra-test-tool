@Library('jenkins-common-library')

//Instanciate Objects from Libs
def util = new libs.utils.Util()

// Parameters to be used on job
properties([
    parameters([
        string(
            name: 'OCP_SERVER',
            defaultValue: 'https://api.yuaxu-maistra-daily.devcluster.openshift.com:6443',
            description: 'OCP Server that will be used'
        ),
        string(
            name: 'IKE_USER',
            defaultValue: 'ike',
            description: 'OCP Server that will be used'
        ),
        password(name: 'IKE_PWD', description: 'Encryption key')
    ])
])

// If the value is empty, so it was triggered by Jenkins, and execution is not needed (only pipeline updates).
if (util.getWhoBuild() == "[]") {
    // Define the build name and informations about it
    currentBuild.displayName = "Not Applicable"
    currentBuild.description = "Triggered Job"

    echo "Nothing to do!"

} else if (OCP_SERVER == "" | IKE_USER == "" | IKE_PWD == ""){
      // Define the build name and informations about it
      currentBuild.displayName = "Not Applicable"
      currentBuild.description = "Need more info"

      echo "Need to inform obrigatory fields!"

} else {

    node('master'){
        // Define the build name and informations about it
        currentBuild.displayName = "${env.BUILD_NUMBER}"
        currentBuild.description = util.htmlDescription(util.whoBuild(util.getWhoBuild()))

        try {
            // Workspace cleanup and git checkout
            gitSteps()
            stage("Create New Project"){
                // Will print the masked value of the KEY, replaced with ****
                wrap([$class: 'MaskPasswordsBuildWrapper', varPasswordPairs: [[var: 'IKE_PWD', password: IKE_PWD]], varMaskRegexes: []]) {
                    sh """
                        #!/bin/bash
                        oc login -u ${params.IKE_USER} -p ${IKE_PWD} --server="${params.OCP_SERVER}" --insecure-skip-tls-verify=true
                        oc new-project maistra-pipelines || true
                    """
                }
            }
            stage("Apply Pipelines"){
                sh """
                    #!/bin/bash
                    cd pipeline
                    oc apply -f openshift-pipeline-subscription.yaml
                    oc apply -f pipeline-cluster-role-binding.yaml
                    oc apply -f pipeline-run-acc-tests.yaml
                """
            }
        } catch(e) {
            currentBuild.result = "FAILED"
            throw e
        } finally {
            stage("Notify Results"){
                catchError(buildResult: 'SUCCESS', stageResult: 'FAILURE') {            
                    // Additional information about the build
                    if (util.getWhoBuild() == "[]") {
                        executedBy = "Jenkins Trigger"
                    } else {
                        executedBy = util.whoBuild(util.getWhoBuild())
                    }                        
                    def moreInfo = "- Executed by: *${executedBy}*"

                    // Slack message to who ran the job
                    slackMessage(currentBuild.result,moreInfo)

                    // Send email to notify
                    emailext body: "<b>Started by:</b> ${executedBy}<p><b>Message:</b> "+ currentBuild.result, subject: "Build number ${env.BUILD_NUMBER} - "+ currentBuild.result, to: "service-mesh-qe@redhat.com"
                }
            } 
        }
    }  
}
