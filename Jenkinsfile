@Library('jenkins-common-library')

//Instanciate Objects from Libs
def util = new libs.utils.Util()

// Parameters to be used on job
properties([
    parameters([
        string(
            name: 'OCP_SERVER',
            defaultValue: '',
            description: 'OCP Server URL'
        ),
        string(
            name: 'IKE_USER',
            defaultValue: '',
            description: 'OCP login user'
        ),
        password(name: 'IKE_PWD', description: 'User password')
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
            stage("Login and Create New Project"){
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
                    sleep 40
                    oc apply -f pipeline-cluster-role-binding.yaml
                """
            }
            stage("Start running all tests"){
                sh """
                    #!/bin/bash
                    oc apply -f pipeline-run-acc-tests.yaml
                    sleep 10
                    podName=$(oc get pods -n maistra-pipelines -l tekton.dev/task=run-all-acc-tests -o jsonpath="{.items[0].metadata.name}")
                """
            }
            stage("check test completed"){
                sh """
                    #!/bin/bash
                    oc logs -n maistra-pipelines ${podName} -c step-run-all-test-cases | grep "#Acc Tests completed#"
                    while [ $? -ne 0 ]; do
                        sleep 60;
                        oc logs -n maistra-pipelines ${podName} -c step-run-all-test-cases | grep "#Acc Tests completed#"
                    done
                """
            }
            stage("collect logs"){
                sh """
                    #!/bin/bash
                    oc cp maistra-pipelines/${podName}:test.log ${WORKSPACE}/tests/test.log -c step-run-all-test-cases
                    oc cp maistra-pipelines/${podName}:results.xml ${WORKSPACE}/tests/results.xml -c step-run-all-test-cases
                """
            }

            post {
                always {
                    archiveArtifacts artifacts: 'tests/results.xml,tests/test.log'
                }
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
