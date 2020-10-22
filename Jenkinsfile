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

        withEnv(["GOPATH=${HOME}/go"]) {
            // Workspace cleanup and git checkout
            gitSteps()
            stage("Login"){
                // Will print the masked value of the KEY, replaced with ****
                wrap([$class: 'MaskPasswordsBuildWrapper', varPasswordPairs: [[var: 'ADMIN_PWD', password: ADMIN_PWD]], varMaskRegexes: []]) {
                    sh """
                        #!/bin/bash
                        oc login -u ${params.ADMIN_USER} -p ${params.ADMIN_PWD} --server="${params.OCP_SERVER}" --insecure-skip-tls-verify=true

                        oc adm policy add-scc-to-user anyuid -z default -n bookinfo
                        oc adm policy add-scc-to-user anyuid -z bookinfo-ratings-v2 -n bookinfo
                        oc adm policy add-scc-to-user anyuid -z httpbin -n bookinfo
                        oc adm policy add-scc-to-user anyuid -z httpbin -n foo
                        oc adm policy add-scc-to-user anyuid -z httpbin -n bar
                        oc adm policy add-scc-to-user anyuid -z httpbin -n legacy

                        mkdir -p ${HOME}/go
                        go get -u github.com/jstemmer/go-junit-report
                    """
                }
            }
        }

         withEnv(["GOPATH=${HOME}/go"]) {
            stage("Start running all tests"){
               sh """
                    #!/bin/bash
                    oc login -u ${params.ADMIN_USER} -p ${params.ADMIN_PWD} --server="${params.OCP_SERVER}" --insecure-skip-tls-verify=true
                    cd tests; go test -timeout 3h -v 2>&1 | tee >(${GOPATH}/bin/go-junit-report > results.xml) test.log
                    set +ex
                    cat ${WORKSPACE}/tests/test.log | grep "FAIL	github.com/Maistra/maistra-test-tool"
                    if [ \$? -eq 0 ]; then
                        currentBuild.result = "FAILED"
                    fi
                    set -ex
                """
            }
        }

            archiveArtifacts artifacts: 'tests/results.xml,tests/test.log'

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
