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
            name: 'ADMIN_USER',
            defaultValue: '',
            description: 'OCP login user'
        ),
        string(
            name: 'BUILD_NAME',
            defaultValue: "${env.BUILD_NUMBER}",
            description: 'currentBuild displayName such as ocp4_4_ossm_1_1_7'
        ),
        password(name: 'ADMIN_PWD', description: 'User password')
    ])
])

// If the value is empty, so it was triggered by Jenkins, and execution is not needed (only pipeline updates).
if (util.getWhoBuild() == "[]") {
    // Define the build name and informations about it
    currentBuild.displayName = "Not Applicable"
    currentBuild.description = "Triggered Job"

    echo "Nothing to do!"

} else if (OCP_SERVER == "" | ADMIN_USER == "" | ADMIN_PWD == ""){
      // Define the build name and informations about it
      currentBuild.displayName = "Not Applicable"
      currentBuild.description = "Need more info"

      echo "Need to inform obrigatory fields!"

} else {

    node('centos'){
        // Define the build name and informations about it
        currentBuild.displayName = "${params.BUILD_NAME}"
        currentBuild.description = util.htmlDescription(util.whoBuild(util.getWhoBuild()))

            // Workspace cleanup and git checkout
            gitSteps()
            stage("Login"){
                // Will print the masked value of the KEY, replaced with ****
                wrap([$class: 'MaskPasswordsBuildWrapper', varPasswordPairs: [[var: 'ADMIN_PWD', password: ADMIN_PWD]], varMaskRegexes: []]) {
                    sh """
                        #!/bin/bash
                        oc login -u ${params.ADMIN_USER} -p ${params.ADMIN_PWD} --server="${params.OCP_SERVER}" --insecure-skip-tls-verify=true
                    """
                }
            }
            stage("Start running all tests"){
                sh "cd tests; go test -run 01 -v"
                sh "cd tests; go test -run 02 -v"
                sh "cd tests; go test -run 03 -v"
                sh "cd tests; go test -run 05 -v"
                sh "cd tests; go test -run 06 -v"
                sh "cd tests; go test -run 07 -v"
                sh "cd tests; go test -run 08 -v"
                sh "cd tests; go test -run 09 -v"
                sh "cd tests; go test -run 10 -v"
                sh "cd tests; go test -run 11 -v"
                sh "cd tests; go test -run 12 -v"
                sh "cd tests; go test -run 13 -v"
                sh "cd tests; go test -run 14 -v"
                sh "cd tests; go test -run 15 -v"
                sh "cd tests; go test -run 16 -v"
                sh "cd tests; go test -run 17 -v"
                sh "cd tests; go test -run 18 -v"
                sh "cd tests; go test -run 19 -v"
                sh "cd tests; go test -run 21 -v"
                sh "cd tests; go test -run 22 -v"
                sh "cd tests; go test -run 24 -v"
                sh "cd tests; go test -run 25 -v"
                sh "cd tests; go test -run 26 -v"
                sh "cd tests; go test -run 27 -v"
                sh "cd tests; go test -run 28 -v"
                sh "cd tests; go test -run 29 -v"
                sh "cd tests; go test -run 30 -v"
            }

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
                    slackMessage(currentBuild.result,moreInfo,currentBuild.displayName)

                    // Send email to notify
                    emailMessage(currentBuild.result,"istio-test@redhat.com", "tests/results.xml,tests/test.log",currentBuild.displayName)
                }
            }
    }  
}
