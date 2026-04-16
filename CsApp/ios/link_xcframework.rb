#!/usr/bin/env ruby
# Fix: re-link CscoreModule files from CsApp/ subdir and add xcframework
require 'xcodeproj'

project_path = File.join(__dir__, 'CsApp.xcodeproj')
project = Xcodeproj::Project.open(project_path)

target = project.targets.find { |t| t.name == 'CsApp' }
raise "CsApp target not found" unless target

group = project.main_group.find_subpath('CsApp', true)

# Remove old references first
group.files.select { |f| f.path =~ /CscoreModule/ }.each do |f|
  target.source_build_phase.files.select { |bf| bf.file_ref == f }.each(&:remove_from_project)
  f.remove_from_project
  puts "Removed old reference: #{f.path}"
end

# Add Swift file (in CsApp/ subdir)
unless group.files.any? { |f| f.path == 'CscoreModule.swift' }
  ref = group.new_file('CscoreModule.swift')
  target.source_build_phase.add_file_reference(ref)
  puts "Added CscoreModule.swift"
end

# Add ObjC file (in CsApp/ subdir)
unless group.files.any? { |f| f.path == 'CscoreModule.m' }
  ref = group.new_file('CscoreModule.m')
  target.source_build_phase.add_file_reference(ref)
  puts "Added CscoreModule.m"
end

# Remove old xcframework references
project.main_group.files.select { |f| f.path&.include?('Cscore.xcframework') }.each do |f|
  target.frameworks_build_phase.files.select { |bf| bf.file_ref == f }.each(&:remove_from_project)
  f.remove_from_project
  puts "Removed old xcframework reference"
end

# Add xcframework
ref = project.main_group.new_file('Cscore.xcframework')
ref.source_tree = 'SOURCE_ROOT'
target.frameworks_build_phase.add_file_reference(ref)

# Embed in app
embed_phase = target.build_phases.find { |p| p.is_a?(Xcodeproj::Project::Object::PBXCopyFilesBuildPhase) && p.name == 'Embed Frameworks' }
unless embed_phase
  embed_phase = project.new(Xcodeproj::Project::Object::PBXCopyFilesBuildPhase)
  embed_phase.name = 'Embed Frameworks'
  embed_phase.symbol_dst_subfolder_spec = :frameworks
  target.build_phases << embed_phase
end
embed_phase.add_file_reference(ref)
puts "Added Cscore.xcframework (linked + embedded)"

# Set framework search path
target.build_configurations.each do |config|
  search_paths = config.build_settings['FRAMEWORK_SEARCH_PATHS'] || ['$(inherited)']
  search_paths = [search_paths] if search_paths.is_a?(String)
  unless search_paths.include?('$(PROJECT_DIR)')
    search_paths << '$(PROJECT_DIR)'
    config.build_settings['FRAMEWORK_SEARCH_PATHS'] = search_paths
  end
end

project.save
puts "Project saved"
