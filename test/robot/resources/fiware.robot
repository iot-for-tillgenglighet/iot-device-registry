*** Settings ***
Library  Collections


*** Keywords ***
Add Device Model Reference
    [Arguments]     ${device}   ${model}
    ${modelid}=         Get From Dictionary  ${model}  id
    ${relationship}=    Create Object Relationship  ${modelid}
    Set To Dictionary   ${device}   refDeviceModel  ${relationship}
    [Return]        ${device}


Create Device
    [Arguments]     ${id}  ${model}=${NONE}
    ${device}=      Create Fiware Entity   id=${id}   type=Device
    ${device}=      Run Keyword If  ${model}    Add Device Model Reference  ${device}   ${model}
    [Return]        ${device}


Create Device Model For Properties
    [Arguments]     ${id}  @{props}
    ${model}=       Create Fiware Entity   id=${id}   type=DeviceModel
    ${category}=    Create Text List Property  sensor
    ${ctrlprops}=   Create Text List Property  @{props}
    Set To Dictionary   ${model}    category  ${category}
    Set To Dictionary   ${model}    controlledProperty  ${ctrlprops}
    [Return]        ${model}


Create Fiware Entity
    [Arguments]     ${type}     ${id}
    @{context}=     Create List
    ...    https://schema.lab.fiware.org/ld/context
    ...    https://uri.etsi.org/ngsi-ld/v1/ngsi-ld-core-context.jsonld
    ${entity}=      Create Dictionary   id=${id}    type=${type}  @context=@{context}
    [Return]        ${entity}


Create Object Relationship
    [Arguments]     ${object}
    ${relationship}=  Create Dictionary  type=Relationship  object=${object}
    [Return]  ${relationship}


Create Text Property
    [Arguments]     ${value}
    ${tp}=      Create Dictionary  type=Property  value=${value}
    [Return]    ${tp}


Create Text List Property
    [Arguments]     @{items}
    @{props}=   Create List  @{items}
    ${tlp}=     Create Dictionary  type=Property  value=@{props}
    [Return]    ${tlp}


Entity Type And ID Should Match
    [Arguments]     ${entity}   ${type}   ${id}
    ${entityID}=    Get From Dictionary     ${entity}     id
    Should Be Equal As Strings      ${entityID}     ${id}

    ${entityType}=    Get From Dictionary     ${entity}     type
    Should Be Equal As Strings      ${entityType}   ${type}


Update Device Value
    [Arguments]     ${session}  ${id}  ${value}
    ${device}=      Create Fiware Entity   id=${id}   type=Device
    ${valueprop}=   Create Text Property  ${value}
    Set To Dictionary   ${device}    value  ${valueprop}
    ${resp}=        PATCH On Session  ${session}  /ngsi-ld/v1/entities/${id}/attrs/  json=${device}
    [Return]        ${resp}
