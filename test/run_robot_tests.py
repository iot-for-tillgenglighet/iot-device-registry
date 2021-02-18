#!/usr/bin/env python3
# -*- coding: utf-8 -*-

'''
A script that sets python paths and imports the needed libs in order to start the robot tests
All test reports that are found are combined into a combined test report at the end.
'''

import os
import glob
import sys
import importlib
import logging

DEPENDENCIES_ROBOT_TEST = {"robot": "robotframework",
                           "RequestsLibrary": "robotframework-requests",
                          }

ROOT = os.path.abspath(os.path.join(os.path.dirname(sys.argv[0]), '..'))

TEST_TOP_DIRECTORY = os.path.join(ROOT, 'test', 'robot', 'testsuites')

def import_and_verify_prerequisites(dependencies):
    """ test that required packages exists """
    missing_imports = []
    logging.debug("verifying required dependencies")
    for lib, text in dependencies.items():
        try:
            importlib.import_module(lib)
        except ImportError:
            logging.error("You need to run 'sudo pip3 install " + text + "'")
            missing_imports.append(text)

    if len(missing_imports):
        raise ImportError("The following imports are missing: " + repr(missing_imports))

#pylint: disable=too-many-arguments
def run(suite_names=None, test_names=None,
        exclude=None, include=None, variables=None, variablefiles=None, non_critical=None,
        output_dir=None, name='robot', dryrun=False):
    '''
    run tests
    '''
    if suite_names is None:
        suite_names = []

    if test_names is None:
        test_names = []

    if exclude is None:
        exclude = []

    if include is None:
        include = []

    if variables is None:
        variables = []

    if variablefiles is None:
        variablefiles = []

    if non_critical is None:
        non_critical = []


    #Add stuff to the python path
    sys.path.append(os.path.join(ROOT, 'test', 'robot', 'libraries'))


    if output_dir is not None:
        opt_output_dir = output_dir
    else:
        opt_output_dir = os.path.join(ROOT, 'test', 'robot', '_output')

    import_and_verify_prerequisites(DEPENDENCIES_ROBOT_TEST)
    import robot

    os.chdir(TEST_TOP_DIRECTORY)
    robot.run(TEST_TOP_DIRECTORY,
                name=name,
                outputdir=opt_output_dir,
                output='%s_results' % name,
                report='%s_report' % name,
                log='%s_log' % name,
                exclude=exclude,
                include=include,
                loglevel='INFO',
                noncritical=non_critical,
                suite=suite_names,
                test=test_names,
                xunit='robottests_%s.xml' % name,
                xunitskipnoncritical=True,
                removekeywords='WUKS',  #remove logging of most failed repeated attempts
                variable=variables,
                variablefile=variablefiles,
                dryrun=dryrun)


def combine_reports(output_dir=None, non_critical=None):
    '''combine all report files that can be found'''
    from robot import rebot_cli

    if non_critical is None:
        non_critical = []

    if output_dir is None:
        output_dir = os.path.join(ROOT, 'test', 'robot', '_output')

    result_file_pattern = os.path.join(output_dir, '*_results.xml')
    result_file_pattern_sub_folders = os.path.join(output_dir, '**', '*_results.xml')

    arguments = ['--name=Combined',
                 '--outputdir=%s' % output_dir,
                 '--output=output.xml',
                 '--loglevel=INFO',
                 '--removekeywords=WUKS']

    if len(non_critical):
        arguments.append('--noncritical=%s' % ' '.join(non_critical))

    if glob.glob(result_file_pattern):
        arguments.append(result_file_pattern)

    if glob.glob(result_file_pattern_sub_folders):
        arguments.append(result_file_pattern_sub_folders)

    rebot_cli(arguments)


def main():
    ''' start robot testing now ! '''
    from argparse import ArgumentParser
    parser = ArgumentParser(description='Runs the automated tests')

    parser.add_argument('--name', default='robot',
                        metavar='NAME', help='overall test suite name to use for this test run')

    parser.add_argument('-n', '--noncritical', default=['not-ready'], nargs='*',
                        metavar='NONCRITICAL_TAGS', help='Select tags that shall be considered non critical')

    parser.add_argument('-e', '--exclude', default=['do-not-run'], nargs='*',
                        metavar='EXCLUDED_TAGS', help='Select tags that shall be excluded')

    parser.add_argument('-i', '--include', default=[], nargs='*',
                        metavar='INCLUDED_TAGS', help='Select tags that shall be excluded')

    parser.add_argument('-s', '--suite-names', default=[], nargs='*',
                        metavar='SUITE_NAME', help='Select a test suite to run')

    parser.add_argument('-t', '--test-names', default=[], nargs='*',
                        metavar='TEST_NAME', help='Select a test to run')

    parser.add_argument('-v', '--variables', default=[], nargs='*',
                        metavar='variable', help='add variables')

    parser.add_argument('-V', '--variablefile', default=[], nargs='*',
                        metavar='variablefile', help='add variable file')

    parser.add_argument('-o', '--outputdir', default=os.path.join(ROOT, 'test', 'robot', '_output'),
                        metavar='OUTPUT_DIR', help='Where to create output files')

    parser.add_argument('--dryrun', action='store_true',
                        help='do not actually call library keywords')

    args = parser.parse_args()


    absolute_variablefiles = []
    for fil in args.variablefile:
        if not os.path.isabs(fil):
            fil = os.path.join(ROOT, fil)

        absolute_variablefiles.append(fil)

    run(name=args.name.replace(' ', '_'),
        suite_names=args.suite_names,
        test_names=args.test_names,
        exclude=args.exclude,
        include=args.include,
        non_critical=args.noncritical,
        variables=args.variables,
        variablefiles=absolute_variablefiles,
        output_dir=args.outputdir,
        dryrun=args.dryrun)

    combine_reports(output_dir=args.outputdir, non_critical=args.noncritical)

if __name__ == '__main__':
    main()
